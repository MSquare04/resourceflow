package demo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"resourceflow/backend/internal/auth"
)

const (
	resetConfirmationValue = "YES"
	accountEmailDomain     = "resourceflow.example"
)

type Summary struct {
	Environment  string
	DatabaseName string
	Accounts     []AccountSummary
	Counts       Counts
}

type AccountSummary struct {
	FullName   string
	Email      string
	Role       string
	Department string
	IsActive   bool
}

type Counts struct {
	Departments    int
	Users          int
	Categories     int
	Types          int
	Rules          int
	Resources      int
	Unavailability int
	Bookings       int
}

type Resetter struct {
	db       *sql.DB
	hasher   auth.PasswordHasher
	now      time.Time
	location *time.Location
}

type departmentSeed struct {
	ID          int64
	Name        string
	Description string
	IsActive    bool
}

type userSeed struct {
	ID           int64
	FullName     string
	Email        string
	DepartmentID int64
	RoleCode     string
	IsActive     bool
}

type categorySeed struct {
	ID          int64
	Code        string
	Name        string
	Description string
	IsActive    bool
}

type resourceTypeSeed struct {
	ID          int64
	CategoryID  int64
	Code        string
	Name        string
	Description string
	IsActive    bool
}

type ruleSeed struct {
	ID                       int64
	ResourceTypeID           int64
	MinDurationMinutes       int
	MaxDurationMinutes       int
	MaxActiveBookingsPerUser int
	RequiresApproval         bool
	BookingHorizonDays       int
	WorkdayStart             string
	WorkdayEnd               string
	UnrestrictedTime         bool
	IsActive                 bool
}

type resourceSeed struct {
	ID           int64
	Name         string
	Description  string
	CategoryID   int64
	TypeID       int64
	DepartmentID *int64
	Location     *string
	Capacity     *int64
	IsBookable   bool
	IsActive     bool
}

type timeWindow struct {
	start time.Time
	end   time.Time
}

func NewResetter(db *sql.DB, hasher auth.PasswordHasher, now time.Time, location *time.Location) *Resetter {
	if location == nil {
		location = time.UTC
	}

	return &Resetter{
		db:       db,
		hasher:   hasher,
		now:      now.UTC().Truncate(time.Second),
		location: location,
	}
}

func ValidateEnvironment(appEnv string) error {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "development", "local":
		return nil
	default:
		return fmt.Errorf("demo reset is allowed only for development/local environment, got %q", appEnv)
	}
}

func ValidateConfirmation() error {
	if strings.TrimSpace(os.Getenv("DEMO_RESET_CONFIRM")) != resetConfirmationValue {
		return fmt.Errorf("set DEMO_RESET_CONFIRM=%s to execute demo reset", resetConfirmationValue)
	}

	return nil
}

func SeedPasswordFromEnv() (string, error) {
	password := strings.TrimSpace(os.Getenv("DEMO_SEED_PASSWORD"))
	if password == "" {
		return "", errors.New("DEMO_SEED_PASSWORD is required")
	}

	return password, nil
}

func (r *Resetter) ResetAndSeed(ctx context.Context, appEnv, dbName, password string) (Summary, error) {
	summary := Summary{
		Environment:  appEnv,
		DatabaseName: dbName,
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return summary, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := resetApplicationData(ctx, tx); err != nil {
		return summary, err
	}

	roleIDs, err := loadRoleIDs(ctx, tx)
	if err != nil {
		return summary, err
	}

	departments, err := seedDepartments(ctx, tx, r.now)
	if err != nil {
		return summary, err
	}

	passwordHash, err := r.hasher.Hash(password)
	if err != nil {
		return summary, fmt.Errorf("hash demo password: %w", err)
	}

	users, accounts, err := seedUsers(ctx, tx, departments, roleIDs, passwordHash, r.now)
	if err != nil {
		return summary, err
	}
	summary.Accounts = accounts

	categories, err := seedResourceCategories(ctx, tx, r.now)
	if err != nil {
		return summary, err
	}

	resourceTypes, err := seedResourceTypes(ctx, tx, categories, r.now)
	if err != nil {
		return summary, err
	}

	if _, err := seedBookingRules(ctx, tx, resourceTypes, r.now); err != nil {
		return summary, err
	}

	resources, err := seedResources(ctx, tx, categories, resourceTypes, departments, r.now)
	if err != nil {
		return summary, err
	}

	if _, err := seedResourceUnavailability(ctx, tx, resources, r.now, r.location); err != nil {
		return summary, err
	}

	if _, err := seedBookings(ctx, tx, resources, users, r.now, r.location); err != nil {
		return summary, err
	}

	if err := tx.Commit(); err != nil {
		return summary, fmt.Errorf("commit demo reset transaction: %w", err)
	}

	counts, err := readCounts(ctx, r.db)
	if err != nil {
		return summary, err
	}
	summary.Counts = counts

	return summary, nil
}

func resetApplicationData(ctx context.Context, tx *sql.Tx) error {
	const query = `
TRUNCATE TABLE
  app.bookings,
  app.resource_unavailability,
  app.resource_availability,
  app.booking_rules,
  app.resources,
  app.resource_types,
  app.resource_categories,
  app.user_roles,
  app.users,
  app.departments
RESTART IDENTITY CASCADE;
`

	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("truncate application tables: %w", err)
	}

	return nil
}

func loadRoleIDs(ctx context.Context, tx *sql.Tx) (map[string]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, code FROM app.roles ORDER BY id;`)
	if err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}
	defer rows.Close()

	roleIDs := make(map[string]int64, 5)
	for rows.Next() {
		var (
			id   int64
			code string
		)
		if err := rows.Scan(&id, &code); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roleIDs[code] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}

	for _, code := range []string{"admin", "manager", "employee", "hr", "interviewer"} {
		if _, ok := roleIDs[code]; !ok {
			return nil, fmt.Errorf("required role %q is missing", code)
		}
	}

	return roleIDs, nil
}

func seedDepartments(ctx context.Context, tx *sql.Tx, now time.Time) (map[string]departmentSeed, error) {
	seeds := []departmentSeed{
		{Name: "Администрация", Description: "Управление и координация корпоративных ресурсов", IsActive: true},
		{Name: "Информационные технологии", Description: "Инфраструктура, сервисы и техническая поддержка", IsActive: true},
		{Name: "Отдел персонала", Description: "Подбор, обучение и сопровождение сотрудников", IsActive: true},
		{Name: "Продажи", Description: "Коммерческие встречи и клиентские демонстрации", IsActive: true},
		{Name: "Эксплуатация", Description: "Транспорт, помещения и техническое обслуживание", IsActive: true},
	}

	result := make(map[string]departmentSeed, len(seeds))
	for _, seed := range seeds {
		seed := seed
		err := tx.QueryRowContext(
			ctx,
			`INSERT INTO app.departments (name, description, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4)
			 RETURNING id;`,
			seed.Name,
			seed.Description,
			seed.IsActive,
			now,
		).Scan(&seed.ID)
		if err != nil {
			return nil, fmt.Errorf("insert department %q: %w", seed.Name, err)
		}
		result[seed.Name] = seed
	}

	return result, nil
}

func seedUsers(
	ctx context.Context,
	tx *sql.Tx,
	departments map[string]departmentSeed,
	roleIDs map[string]int64,
	passwordHash string,
	now time.Time,
) (map[string]userSeed, []AccountSummary, error) {
	seeds := []struct {
		fullName   string
		localPart  string
		department string
		roleCode   string
		isActive   bool
	}{
		{fullName: "Анна Смирнова", localPart: "anna.smirnova", department: "Администрация", roleCode: "admin", isActive: true},
		{fullName: "Михаил Волков", localPart: "mikhail.volkov", department: "Эксплуатация", roleCode: "manager", isActive: true},
		{fullName: "Елена Кузнецова", localPart: "elena.kuznetsova", department: "Информационные технологии", roleCode: "employee", isActive: true},
		{fullName: "Ольга Петрова", localPart: "olga.petrova", department: "Отдел персонала", roleCode: "hr", isActive: true},
		{fullName: "Алексей Орлов", localPart: "alexey.orlov", department: "Продажи", roleCode: "interviewer", isActive: true},
		{fullName: "Игорь Соколов", localPart: "igor.sokolov", department: "Информационные технологии", roleCode: "employee", isActive: false},
	}

	users := make(map[string]userSeed, len(seeds))
	accounts := make([]AccountSummary, 0, len(seeds))

	for _, seed := range seeds {
		department, ok := departments[seed.department]
		if !ok {
			return nil, nil, fmt.Errorf("department %q is not seeded", seed.department)
		}

		roleID, ok := roleIDs[seed.roleCode]
		if !ok {
			return nil, nil, fmt.Errorf("role %q is not seeded", seed.roleCode)
		}

		user := userSeed{
			FullName:     seed.fullName,
			Email:        fmt.Sprintf("%s@%s", seed.localPart, accountEmailDomain),
			DepartmentID: department.ID,
			RoleCode:     seed.roleCode,
			IsActive:     seed.isActive,
		}

		err := tx.QueryRowContext(
			ctx,
			`INSERT INTO app.users (full_name, email, password_hash, department_id, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6)
			 RETURNING id;`,
			user.FullName,
			user.Email,
			passwordHash,
			user.DepartmentID,
			user.IsActive,
			now,
		).Scan(&user.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("insert user %q: %w", user.Email, err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO app.user_roles (user_id, role_id) VALUES ($1, $2);`,
			user.ID,
			roleID,
		); err != nil {
			return nil, nil, fmt.Errorf("assign role %q to %q: %w", seed.roleCode, user.Email, err)
		}

		users[user.Email] = user
		accounts = append(accounts, AccountSummary{
			FullName:   user.FullName,
			Email:      user.Email,
			Role:       seed.roleCode,
			Department: seed.department,
			IsActive:   user.IsActive,
		})
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Email < accounts[j].Email
	})

	return users, accounts, nil
}

func seedResourceCategories(ctx context.Context, tx *sql.Tx, now time.Time) (map[string]categorySeed, error) {
	seeds := []categorySeed{
		{Code: "rooms", Name: "Помещения", Description: "Комнаты, залы и другие бронируемые пространства", IsActive: true},
		{Code: "transport", Name: "Транспорт", Description: "Корпоративный транспорт для служебных поездок", IsActive: true},
		{Code: "equipment", Name: "Оборудование", Description: "Техника и переносимые устройства", IsActive: true},
		{Code: "workplaces", Name: "Рабочие места", Description: "Фиксированные рабочие места и посадочные зоны", IsActive: true},
	}

	result := make(map[string]categorySeed, len(seeds))
	for _, seed := range seeds {
		seed := seed
		err := tx.QueryRowContext(
			ctx,
			`INSERT INTO app.resource_categories (code, name, description, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $5)
			 RETURNING id;`,
			seed.Code,
			seed.Name,
			seed.Description,
			seed.IsActive,
			now,
		).Scan(&seed.ID)
		if err != nil {
			return nil, fmt.Errorf("insert resource category %q: %w", seed.Code, err)
		}
		result[seed.Code] = seed
	}

	return result, nil
}

func seedResourceTypes(
	ctx context.Context,
	tx *sql.Tx,
	categories map[string]categorySeed,
	now time.Time,
) (map[string]resourceTypeSeed, error) {
	seeds := []struct {
		code        string
		category    string
		name        string
		description string
		isActive    bool
	}{
		{code: "meeting-room", category: "rooms", name: "Переговорная", description: "Небольшая комната для встреч и интервью", isActive: true},
		{code: "conference-hall", category: "rooms", name: "Конференц-зал", description: "Большой зал для презентаций и общих встреч", isActive: true},
		{code: "passenger-car", category: "transport", name: "Легковой автомобиль", description: "Служебный автомобиль для поездок сотрудников", isActive: true},
		{code: "cargo-van", category: "transport", name: "Грузовой автомобиль", description: "Транспорт для перевозки оборудования и хозяйственных грузов", isActive: true},
		{code: "minibus", category: "transport", name: "Микроавтобус", description: "Транспорт для групповых поездок и выездных мероприятий", isActive: true},
		{code: "projector", category: "equipment", name: "Проектор", description: "Переносной проектор для встреч и обучения", isActive: true},
		{code: "laptop", category: "equipment", name: "Ноутбук", description: "Мобильное рабочее устройство для презентаций и выездной работы", isActive: true},
		{code: "workspace", category: "workplaces", name: "Рабочее место", description: "Фиксированное место в офисном пространстве", isActive: true},
	}

	result := make(map[string]resourceTypeSeed, len(seeds))
	for _, seed := range seeds {
		category, ok := categories[seed.category]
		if !ok {
			return nil, fmt.Errorf("resource category %q is not seeded", seed.category)
		}

		resourceType := resourceTypeSeed{
			CategoryID:  category.ID,
			Code:        seed.code,
			Name:        seed.name,
			Description: seed.description,
			IsActive:    seed.isActive,
		}

		err := tx.QueryRowContext(
			ctx,
			`INSERT INTO app.resource_types (category_id, code, name, description, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6)
			 RETURNING id;`,
			resourceType.CategoryID,
			resourceType.Code,
			resourceType.Name,
			resourceType.Description,
			resourceType.IsActive,
			now,
		).Scan(&resourceType.ID)
		if err != nil {
			return nil, fmt.Errorf("insert resource type %q: %w", resourceType.Code, err)
		}
		result[resourceType.Code] = resourceType
	}

	return result, nil
}

func seedBookingRules(
	ctx context.Context,
	tx *sql.Tx,
	resourceTypes map[string]resourceTypeSeed,
	now time.Time,
) (map[string]ruleSeed, error) {
	seeds := []struct {
		typeCode string
		rule     ruleSeed
	}{
		{typeCode: "meeting-room", rule: ruleSeed{ResourceTypeID: resourceTypes["meeting-room"].ID, MinDurationMinutes: 30, MaxDurationMinutes: 180, MaxActiveBookingsPerUser: 20, RequiresApproval: false, BookingHorizonDays: 30, WorkdayStart: "09:00", WorkdayEnd: "18:00", UnrestrictedTime: false, IsActive: true}},
		{typeCode: "conference-hall", rule: ruleSeed{ResourceTypeID: resourceTypes["conference-hall"].ID, MinDurationMinutes: 60, MaxDurationMinutes: 480, MaxActiveBookingsPerUser: 20, RequiresApproval: true, BookingHorizonDays: 60, WorkdayStart: "10:00", WorkdayEnd: "19:00", UnrestrictedTime: false, IsActive: true}},
		{typeCode: "passenger-car", rule: ruleSeed{ResourceTypeID: resourceTypes["passenger-car"].ID, MinDurationMinutes: 60, MaxDurationMinutes: 480, MaxActiveBookingsPerUser: 20, RequiresApproval: true, BookingHorizonDays: 14, WorkdayStart: "07:00", WorkdayEnd: "22:00", UnrestrictedTime: false, IsActive: true}},
		{typeCode: "cargo-van", rule: ruleSeed{ResourceTypeID: resourceTypes["cargo-van"].ID, MinDurationMinutes: 60, MaxDurationMinutes: 600, MaxActiveBookingsPerUser: 20, RequiresApproval: true, BookingHorizonDays: 10, WorkdayStart: "06:00", WorkdayEnd: "20:00", UnrestrictedTime: false, IsActive: true}},
		{typeCode: "minibus", rule: ruleSeed{ResourceTypeID: resourceTypes["minibus"].ID, MinDurationMinutes: 60, MaxDurationMinutes: 720, MaxActiveBookingsPerUser: 20, RequiresApproval: true, BookingHorizonDays: 21, WorkdayStart: "06:00", WorkdayEnd: "22:00", UnrestrictedTime: false, IsActive: true}},
		{typeCode: "projector", rule: ruleSeed{ResourceTypeID: resourceTypes["projector"].ID, MinDurationMinutes: 30, MaxDurationMinutes: 480, MaxActiveBookingsPerUser: 20, RequiresApproval: false, BookingHorizonDays: 30, WorkdayStart: "00:00", WorkdayEnd: "23:59", UnrestrictedTime: true, IsActive: true}},
		{typeCode: "laptop", rule: ruleSeed{ResourceTypeID: resourceTypes["laptop"].ID, MinDurationMinutes: 5, MaxDurationMinutes: 600, MaxActiveBookingsPerUser: 20, RequiresApproval: false, BookingHorizonDays: 14, WorkdayStart: "00:00", WorkdayEnd: "23:59", UnrestrictedTime: true, IsActive: true}},
		{typeCode: "workspace", rule: ruleSeed{ResourceTypeID: resourceTypes["workspace"].ID, MinDurationMinutes: 60, MaxDurationMinutes: 480, MaxActiveBookingsPerUser: 20, RequiresApproval: false, BookingHorizonDays: 14, WorkdayStart: "09:00", WorkdayEnd: "18:00", UnrestrictedTime: false, IsActive: false}},
	}

	result := make(map[string]ruleSeed, len(seeds))
	for _, item := range seeds {
		seed := item.rule
		err := tx.QueryRowContext(
			ctx,
			`INSERT INTO app.booking_rules (
			    resource_type_id,
			    min_duration_minutes,
			    max_duration_minutes,
			    max_active_bookings_per_user,
			    requires_approval,
			    booking_horizon_days,
			    workday_start,
			    workday_end,
			    unrestricted_time,
			    is_active,
			    created_at,
			    updated_at
			  )
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
			  RETURNING id;`,
			seed.ResourceTypeID,
			seed.MinDurationMinutes,
			seed.MaxDurationMinutes,
			seed.MaxActiveBookingsPerUser,
			seed.RequiresApproval,
			seed.BookingHorizonDays,
			seed.WorkdayStart,
			seed.WorkdayEnd,
			seed.UnrestrictedTime,
			seed.IsActive,
			now,
		).Scan(&seed.ID)
		if err != nil {
			return nil, fmt.Errorf("insert booking rule for %q: %w", item.typeCode, err)
		}
		result[item.typeCode] = seed
	}

	return result, nil
}

func seedResources(
	ctx context.Context,
	tx *sql.Tx,
	categories map[string]categorySeed,
	resourceTypes map[string]resourceTypeSeed,
	departments map[string]departmentSeed,
	now time.Time,
) (map[string]resourceSeed, error) {
	departmentPtr := func(name string) *int64 {
		id := departments[name].ID
		return &id
	}
	int64Ptr := func(value int64) *int64 {
		v := value
		return &v
	}

	seeds := []struct {
		key         string
		name        string
		description string
		category    string
		typeCode    string
		department  *int64
		location    *string
		capacity    *int64
		isBookable  bool
		isActive    bool
	}{
		{
			key:         "neva",
			name:        "Переговорная «Нева»",
			description: "Светлая переговорная рядом с техническим блоком для встреч, интервью и ежедневных синхронизаций.",
			category:    "rooms",
			typeCode:    "meeting-room",
			department:  departmentPtr("Информационные технологии"),
			location:    stringPtr("Офис, 3 этаж, блок B"),
			capacity:    int64Ptr(8),
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "volga",
			name:        "Конференц-зал «Волга»",
			description: "Основной зал для квартальных презентаций, общих встреч и защиты проектных инициатив.",
			category:    "rooms",
			typeCode:    "conference-hall",
			department:  departmentPtr("Администрация"),
			location:    stringPtr("Офис, 1 этаж, центральный холл"),
			capacity:    int64Ptr(24),
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "skoda-octavia",
			name:        "Автомобиль Skoda Octavia",
			description: "Служебный автомобиль для выездов к клиентам и поездок между офисами.",
			category:    "transport",
			typeCode:    "passenger-car",
			department:  departmentPtr("Эксплуатация"),
			location:    stringPtr("Парковка B, место 12"),
			capacity:    int64Ptr(5),
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "hyundai-solaris",
			name:        "Автомобиль Hyundai Solaris",
			description: "Резервный автомобиль. Бронирование временно отключено до планового техобслуживания.",
			category:    "transport",
			typeCode:    "passenger-car",
			department:  departmentPtr("Эксплуатация"),
			location:    stringPtr("Парковка A, место 4"),
			capacity:    int64Ptr(5),
			isBookable:  false,
			isActive:    true,
		},
		{
			key:         "gazel-next",
			name:        "ГАЗель Next",
			description: "Грузовой автомобиль для перевозки оборудования, мебели и хозяйственных материалов.",
			category:    "transport",
			typeCode:    "cargo-van",
			department:  departmentPtr("Эксплуатация"),
			location:    stringPtr("Грузовая парковка, место G2"),
			capacity:    int64Ptr(2),
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "ford-transit",
			name:        "Ford Transit",
			description: "Микроавтобус для групповых выездов на обучение, собеседования и клиентские мероприятия.",
			category:    "transport",
			typeCode:    "minibus",
			department:  departmentPtr("Эксплуатация"),
			location:    stringPtr("Парковка C, место 2"),
			capacity:    int64Ptr(8),
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "epson-eb-fh52",
			name:        "Проектор Epson EB-FH52",
			description: "Портативный проектор для внутренних презентаций, обучения и переговоров с демонстрацией материалов.",
			category:    "equipment",
			typeCode:    "projector",
			department:  departmentPtr("Отдел персонала"),
			location:    stringPtr("Склад техники, шкаф 2"),
			capacity:    nil,
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "thinkpad-t14",
			name:        "Lenovo ThinkPad T14",
			description: "Ноутбук для выездной работы, презентаций и оперативной замены стационарного рабочего места.",
			category:    "equipment",
			typeCode:    "laptop",
			department:  departmentPtr("Информационные технологии"),
			location:    stringPtr("Выдача техники, стойка 1"),
			capacity:    nil,
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "workspace-a17",
			name:        "Рабочее место A-17",
			description: "Фиксированное рабочее место у окна в open space отдела продаж.",
			category:    "workplaces",
			typeCode:    "workspace",
			department:  departmentPtr("Продажи"),
			location:    stringPtr("Офис, 2 этаж, зона A"),
			capacity:    int64Ptr(1),
			isBookable:  true,
			isActive:    true,
		},
		{
			key:         "ladoga",
			name:        "Переговорная «Ладога»",
			description: "Комната временно выведена из эксплуатации на время ремонта и перенастройки вентиляции.",
			category:    "rooms",
			typeCode:    "meeting-room",
			department:  departmentPtr("Администрация"),
			location:    stringPtr("Офис, 4 этаж, блок C"),
			capacity:    int64Ptr(6),
			isBookable:  true,
			isActive:    false,
		},
	}

	result := make(map[string]resourceSeed, len(seeds))
	for _, seed := range seeds {
		category, ok := categories[seed.category]
		if !ok {
			return nil, fmt.Errorf("resource category %q is not seeded", seed.category)
		}
		resourceType, ok := resourceTypes[seed.typeCode]
		if !ok {
			return nil, fmt.Errorf("resource type %q is not seeded", seed.typeCode)
		}

		resource := resourceSeed{
			Name:         seed.name,
			Description:  seed.description,
			CategoryID:   category.ID,
			TypeID:       resourceType.ID,
			DepartmentID: seed.department,
			Location:     seed.location,
			Capacity:     seed.capacity,
			IsBookable:   seed.isBookable,
			IsActive:     seed.isActive,
		}

		err := tx.QueryRowContext(
			ctx,
			`INSERT INTO app.resources (
			    name, description, category_id, type_id, department_id, location, capacity, is_bookable, is_active, created_at, updated_at
			  )
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
			  RETURNING id;`,
			resource.Name,
			resource.Description,
			resource.CategoryID,
			resource.TypeID,
			nullableInt64(resource.DepartmentID),
			nullableString(resource.Location),
			nullableInt64(resource.Capacity),
			resource.IsBookable,
			resource.IsActive,
			now,
		).Scan(&resource.ID)
		if err != nil {
			return nil, fmt.Errorf("insert resource %q: %w", seed.name, err)
		}

		result[seed.key] = resource
	}

	return result, nil
}

func seedResourceUnavailability(
	ctx context.Context,
	tx *sql.Tx,
	resources map[string]resourceSeed,
	now time.Time,
	location *time.Location,
) (int, error) {
	dayStart := localDayStart(now, location)
	startFromTomorrow := dayStart.AddDate(0, 0, 1)
	pastDay := dayStart.AddDate(0, 0, -2)

	intervalsByResource := map[string][]struct {
		window timeWindow
		reason *string
	}{
		"neva":          buildRepeatedUnavailability(startFromTomorrow, 6, 13, 0, 15, 0, "Временное закрытие переговорной для настройки оборудования"),
		"gazel-next":    buildRepeatedUnavailability(startFromTomorrow.AddDate(0, 0, 1), 5, 12, 0, 14, 0, "Техническое обслуживание автомобиля"),
		"epson-eb-fh52": buildRepeatedUnavailability(startFromTomorrow.AddDate(0, 0, 2), 4, 16, 0, 18, 0, "Ремонт оборудования"),
	}

	intervalsByResource["neva"] = append(intervalsByResource["neva"], struct {
		window timeWindow
		reason *string
	}{
		window: timeWindow{start: dayAt(pastDay, 14, 0), end: dayAt(pastDay, 15, 0)},
		reason: stringPtr("Завершённое временное закрытие переговорной"),
	})
	intervalsByResource["gazel-next"] = append(intervalsByResource["gazel-next"], struct {
		window timeWindow
		reason *string
	}{
		window: timeWindow{start: dayAt(pastDay, 11, 0), end: dayAt(pastDay, 13, 0)},
		reason: stringPtr("Завершённое обслуживание автомобиля"),
	})
	intervalsByResource["epson-eb-fh52"] = append(intervalsByResource["epson-eb-fh52"], struct {
		window timeWindow
		reason *string
	}{
		window: timeWindow{start: dayAt(pastDay, 9, 0), end: dayAt(pastDay, 10, 0)},
		reason: stringPtr("Завершённый ремонт проектора"),
	})

	total := 0
	for key, intervals := range intervalsByResource {
		resource, ok := resources[key]
		if !ok {
			return 0, fmt.Errorf("resource %q is not seeded", key)
		}

		for _, interval := range intervals {
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO app.resource_unavailability (resource_id, start_at, end_at, reason, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $5);`,
				resource.ID,
				interval.window.start,
				interval.window.end,
				nullableString(interval.reason),
				now,
			); err != nil {
				return 0, fmt.Errorf("insert resource unavailability for %q: %w", resource.Name, err)
			}
			total++
		}
	}

	return total, nil
}

func seedBookings(
	ctx context.Context,
	tx *sql.Tx,
	resources map[string]resourceSeed,
	users map[string]userSeed,
	now time.Time,
	location *time.Location,
) (int, error) {
	dayStart := localDayStart(now, location)
	tomorrow := dayStart.AddDate(0, 0, 1)
	day2 := tomorrow.AddDate(0, 0, 1)
	day3 := tomorrow.AddDate(0, 0, 2)
	day4 := tomorrow.AddDate(0, 0, 3)
	day5 := tomorrow.AddDate(0, 0, 4)
	day6 := tomorrow.AddDate(0, 0, 5)
	day7 := tomorrow.AddDate(0, 0, 6)
	pastDay := dayStart.AddDate(0, 0, -2)

	admin := users["anna.smirnova@resourceflow.example"]
	manager := users["mikhail.volkov@resourceflow.example"]
	employee := users["elena.kuznetsova@resourceflow.example"]
	hr := users["olga.petrova@resourceflow.example"]
	interviewer := users["alexey.orlov@resourceflow.example"]

	seeds := []struct {
		resourceKey      string
		userID           int64
		startAt          time.Time
		endAt            time.Time
		purpose          *string
		status           string
		approvedByUserID *int64
		approvedAt       *time.Time
		cancelledAt      *time.Time
		completedAt      *time.Time
		createdAt        time.Time
		updatedAt        time.Time
	}{
		{
			resourceKey: "volga",
			userID:      employee.ID,
			startAt:     dayAt(tomorrow, 10, 0),
			endAt:       dayAt(tomorrow, 12, 0),
			purpose:     stringPtr("Командная презентация квартального плана"),
			status:      "pending",
			createdAt:   now.Add(-6 * time.Hour),
			updatedAt:   now.Add(-6 * time.Hour),
		},
		{
			resourceKey: "skoda-octavia",
			userID:      hr.ID,
			startAt:     dayAt(day2, 9, 0),
			endAt:       dayAt(day2, 13, 0),
			purpose:     stringPtr("Выезд на ярмарку вакансий"),
			status:      "pending",
			createdAt:   now.Add(-5 * time.Hour),
			updatedAt:   now.Add(-5 * time.Hour),
		},
		{
			resourceKey: "gazel-next",
			userID:      manager.ID,
			startAt:     dayAt(day3, 8, 0),
			endAt:       dayAt(day3, 11, 0),
			purpose:     stringPtr("Перевозка оборудования на выездное мероприятие"),
			status:      "pending",
			createdAt:   now.Add(-4 * time.Hour),
			updatedAt:   now.Add(-4 * time.Hour),
		},
		{
			resourceKey: "neva",
			userID:      employee.ID,
			startAt:     dayAt(tomorrow, 10, 0),
			endAt:       dayAt(tomorrow, 11, 0),
			purpose:     stringPtr("Технический разбор инцидента"),
			status:      "confirmed",
			createdAt:   now.Add(-3 * time.Hour),
			updatedAt:   now.Add(-3 * time.Hour),
		},
		{
			resourceKey: "epson-eb-fh52",
			userID:      admin.ID,
			startAt:     dayAt(day3, 11, 0),
			endAt:       dayAt(day3, 13, 0),
			purpose:     stringPtr("Демонстрация материалов для стратегической сессии"),
			status:      "confirmed",
			createdAt:   now.Add(-170 * time.Minute),
			updatedAt:   now.Add(-170 * time.Minute),
		},
		{
			resourceKey:      "volga",
			userID:           admin.ID,
			startAt:          dayAt(day6, 10, 0),
			endAt:            dayAt(day6, 13, 0),
			purpose:          stringPtr("Защита инициатив по бюджету"),
			status:           "confirmed",
			approvedByUserID: &manager.ID,
			approvedAt:       timePtr(now.Add(-90 * time.Minute)),
			createdAt:        now.Add(-95 * time.Minute),
			updatedAt:        now.Add(-90 * time.Minute),
		},
		{
			resourceKey:      "ford-transit",
			userID:           manager.ID,
			startAt:          dayAt(day4, 8, 0),
			endAt:            dayAt(day4, 10, 0),
			purpose:          stringPtr("Трансфер команды на обучение в филиал"),
			status:           "confirmed",
			approvedByUserID: &admin.ID,
			approvedAt:       timePtr(now.Add(-80 * time.Minute)),
			createdAt:        now.Add(-85 * time.Minute),
			updatedAt:        now.Add(-80 * time.Minute),
		},
		{
			resourceKey: "thinkpad-t14",
			userID:      interviewer.ID,
			startAt:     dayAt(day3, 15, 0),
			endAt:       dayAt(day3, 16, 0),
			purpose:     stringPtr("Ежедневное интервью с кандидатами"),
			status:      "confirmed",
			createdAt:   now.Add(-70 * time.Minute),
			updatedAt:   now.Add(-70 * time.Minute),
		},
		{
			resourceKey: "thinkpad-t14",
			userID:      interviewer.ID,
			startAt:     dayAt(day4, 15, 0),
			endAt:       dayAt(day4, 16, 0),
			purpose:     stringPtr("Ежедневное интервью с кандидатами"),
			status:      "confirmed",
			createdAt:   now.Add(-65 * time.Minute),
			updatedAt:   now.Add(-65 * time.Minute),
		},
		{
			resourceKey: "thinkpad-t14",
			userID:      interviewer.ID,
			startAt:     dayAt(day5, 15, 0),
			endAt:       dayAt(day5, 16, 0),
			purpose:     stringPtr("Ежедневное интервью с кандидатами"),
			status:      "confirmed",
			createdAt:   now.Add(-60 * time.Minute),
			updatedAt:   now.Add(-60 * time.Minute),
		},
		{
			resourceKey: "neva",
			userID:      manager.ID,
			startAt:     dayAt(day7, 16, 0),
			endAt:       dayAt(day7, 17, 0),
			purpose:     stringPtr("Встреча по координации ремонта этажей"),
			status:      "confirmed",
			createdAt:   now.Add(-50 * time.Minute),
			updatedAt:   now.Add(-50 * time.Minute),
		},
		{
			resourceKey: "neva",
			userID:      manager.ID,
			startAt:     dayAt(pastDay, 10, 0),
			endAt:       dayAt(pastDay, 11, 0),
			purpose:     stringPtr("Итоговое собеседование с подрядчиком"),
			status:      "completed",
			completedAt: timePtr(dayAt(pastDay, 11, 5)),
			createdAt:   dayAt(pastDay, 8, 0),
			updatedAt:   dayAt(pastDay, 11, 5),
		},
		{
			resourceKey:      "volga",
			userID:           hr.ID,
			startAt:          dayAt(day5, 14, 0),
			endAt:            dayAt(day5, 16, 0),
			purpose:          stringPtr("Обучение команды адаптации"),
			status:           "rejected",
			approvedByUserID: &admin.ID,
			approvedAt:       timePtr(now.Add(-75 * time.Minute)),
			createdAt:        now.Add(-80 * time.Minute),
			updatedAt:        now.Add(-75 * time.Minute),
		},
		{
			resourceKey: "skoda-octavia",
			userID:      employee.ID,
			startAt:     dayAt(day5, 9, 0),
			endAt:       dayAt(day5, 11, 0),
			purpose:     stringPtr("Отмена поездки в филиал"),
			status:      "cancelled",
			cancelledAt: timePtr(now.Add(-30 * time.Minute)),
			createdAt:   now.Add(-45 * time.Minute),
			updatedAt:   now.Add(-30 * time.Minute),
		},
	}

	for _, seed := range seeds {
		resource, ok := resources[seed.resourceKey]
		if !ok {
			return 0, fmt.Errorf("resource %q is not seeded", seed.resourceKey)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO app.bookings (
			    resource_id, user_id, start_at, end_at, purpose, status,
			    approved_by_user_id, approved_at, cancelled_at, completed_at, created_at, updated_at
			  )
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);`,
			resource.ID,
			seed.userID,
			seed.startAt,
			seed.endAt,
			nullableString(seed.purpose),
			seed.status,
			nullableInt64(seed.approvedByUserID),
			nullableTime(seed.approvedAt),
			nullableTime(seed.cancelledAt),
			nullableTime(seed.completedAt),
			seed.createdAt,
			seed.updatedAt,
		); err != nil {
			return 0, fmt.Errorf("insert booking for resource %q: %w", resource.Name, err)
		}
	}

	return len(seeds), nil
}

func readCounts(ctx context.Context, db *sql.DB) (Counts, error) {
	type tableCount struct {
		target *int
		table  string
	}

	counts := Counts{}
	queries := []tableCount{
		{target: &counts.Departments, table: "app.departments"},
		{target: &counts.Users, table: "app.users"},
		{target: &counts.Categories, table: "app.resource_categories"},
		{target: &counts.Types, table: "app.resource_types"},
		{target: &counts.Rules, table: "app.booking_rules"},
		{target: &counts.Resources, table: "app.resources"},
		{target: &counts.Unavailability, table: "app.resource_unavailability"},
		{target: &counts.Bookings, table: "app.bookings"},
	}

	for _, item := range queries {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", item.table)
		if err := db.QueryRowContext(ctx, query).Scan(item.target); err != nil {
			return Counts{}, fmt.Errorf("count rows in %s: %w", item.table, err)
		}
	}

	return counts, nil
}

func buildRepeatedUnavailability(
	startDay time.Time,
	days int,
	startHour, startMinute, endHour, endMinute int,
	reason string,
) []struct {
	window timeWindow
	reason *string
} {
	result := make([]struct {
		window timeWindow
		reason *string
	}, 0, days)

	for dayOffset := 0; dayOffset < days; dayOffset++ {
		currentDay := startDay.AddDate(0, 0, dayOffset)
		result = append(result, struct {
			window timeWindow
			reason *string
		}{
			window: timeWindow{
				start: dayAt(currentDay, startHour, startMinute),
				end:   dayAt(currentDay, endHour, endMinute),
			},
			reason: stringPtr(reason),
		})
	}

	return result
}

func localDayStart(now time.Time, location *time.Location) time.Time {
	if location == nil {
		location = time.UTC
	}

	current := now.In(location)
	return time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, location)
}

func dayAt(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location()).UTC()
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func stringPtr(value string) *string {
	v := value
	return &v
}

func timePtr(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}
