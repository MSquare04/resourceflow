package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"resourceflow/backend/internal/model"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int64) (model.User, error)
	Create(ctx context.Context, params CreateUserParams) (model.User, error)
	List(ctx context.Context) ([]model.User, error)
	Update(ctx context.Context, id int64, params UpdateUserParams) (model.User, error)
	UpdatePassword(ctx context.Context, id int64, params UpdateUserPasswordParams) (model.User, error)
	ListRolesByUserID(ctx context.Context, userID int64) ([]string, error)
	ValidateRoleCodes(ctx context.Context, roleCodes []string) error
	ReplaceRolesByUserID(ctx context.Context, userID int64, roleCodes []string) error
}

type PostgresUserRepository struct {
	db *sql.DB
}

var ErrRoleNotFound = errors.New("role code not found")

type CreateUserParams struct {
	FullName     string
	Email        string
	PasswordHash string
	DepartmentID *int64
	IsActive     bool
}

type UpdateUserParams struct {
	FullName     string
	Email        string
	PasswordHash *string
	DepartmentID *int64
	IsActive     bool
}

type UpdateUserPasswordParams struct {
	PasswordHash string
}

func NewUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	query := `
SELECT id, full_name, email, password_hash, auth_version, department_id, is_active
FROM app.users
WHERE lower(email) = lower($1)
LIMIT 1;
`

	var user model.User
	var departmentID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.AuthVersion,
		&departmentID,
		&user.IsActive,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("find user by email query failed: %w", err)
	}
	if departmentID.Valid {
		user.DepartmentID = &departmentID.Int64
	}

	return user, nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id int64) (model.User, error) {
	query := `
SELECT id, full_name, email, password_hash, auth_version, department_id, is_active
FROM app.users
WHERE id = $1
LIMIT 1;
`

	var user model.User
	var departmentID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.AuthVersion,
		&departmentID,
		&user.IsActive,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("find user by id query failed: %w", err)
	}
	if departmentID.Valid {
		user.DepartmentID = &departmentID.Int64
	}

	return user, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, params CreateUserParams) (model.User, error) {
	query := `
INSERT INTO app.users (full_name, email, password_hash, department_id, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, full_name, email, password_hash, auth_version, department_id, is_active;
`

	var user model.User
	var departmentID sql.NullInt64
	err := r.db.QueryRowContext(
		ctx,
		query,
		params.FullName,
		params.Email,
		params.PasswordHash,
		nullableInt64(params.DepartmentID),
		params.IsActive,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.AuthVersion,
		&departmentID,
		&user.IsActive,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("create user query failed: %w", err)
	}
	if departmentID.Valid {
		user.DepartmentID = &departmentID.Int64
	}

	return user, nil
}

func (r *PostgresUserRepository) List(ctx context.Context) ([]model.User, error) {
	query := `
SELECT id, full_name, email, password_hash, auth_version, department_id, is_active
FROM app.users
ORDER BY id;
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users query failed: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		var departmentID sql.NullInt64
		if err := rows.Scan(&user.ID, &user.FullName, &user.Email, &user.PasswordHash, &user.AuthVersion, &departmentID, &user.IsActive); err != nil {
			return nil, fmt.Errorf("list users scan failed: %w", err)
		}
		if departmentID.Valid {
			user.DepartmentID = &departmentID.Int64
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users rows failed: %w", err)
	}

	return users, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, id int64, params UpdateUserParams) (model.User, error) {
	query := `
UPDATE app.users
SET full_name = $2,
    email = $3,
    password_hash = CASE WHEN $4::text IS NULL THEN password_hash ELSE $4 END,
    department_id = $5,
    is_active = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING id, full_name, email, password_hash, auth_version, department_id, is_active;
`

	var user model.User
	var departmentID sql.NullInt64
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
		params.FullName,
		params.Email,
		nullableString(params.PasswordHash),
		nullableInt64(params.DepartmentID),
		params.IsActive,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.AuthVersion,
		&departmentID,
		&user.IsActive,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("update user query failed: %w", err)
	}
	if departmentID.Valid {
		user.DepartmentID = &departmentID.Int64
	}

	return user, nil
}

func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id int64, params UpdateUserPasswordParams) (model.User, error) {
	query := `
UPDATE app.users
SET password_hash = $2,
    auth_version = auth_version + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING id, full_name, email, password_hash, auth_version, department_id, is_active;
`

	var user model.User
	var departmentID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, id, params.PasswordHash).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.AuthVersion,
		&departmentID,
		&user.IsActive,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("update user password query failed: %w", err)
	}
	if departmentID.Valid {
		user.DepartmentID = &departmentID.Int64
	}

	return user, nil
}

func (r *PostgresUserRepository) ListRolesByUserID(ctx context.Context, userID int64) ([]string, error) {
	query := `
SELECT r.code
FROM app.user_roles ur
INNER JOIN app.roles r ON r.id = ur.role_id
WHERE ur.user_id = $1
ORDER BY r.code;
`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user roles query failed: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("list user roles scan failed: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user roles rows failed: %w", err)
	}

	return roles, nil
}

func (r *PostgresUserRepository) ReplaceRolesByUserID(ctx context.Context, userID int64, roleCodes []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace user roles transaction failed: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM app.users WHERE id = $1);`, userID).Scan(&exists); err != nil {
		return fmt.Errorf("check user exists for role replace failed: %w", err)
	}
	if !exists {
		return sql.ErrNoRows
	}

	roleIDs, err := resolveRoleIDs(ctx, tx, roleCodes)
	if err != nil {
		return fmt.Errorf("resolve role ids failed: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM app.user_roles WHERE user_id = $1;`, userID); err != nil {
		return fmt.Errorf("delete existing user roles failed: %w", err)
	}

	for _, roleID := range roleIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO app.user_roles (user_id, role_id) VALUES ($1, $2);`, userID, roleID); err != nil {
			return fmt.Errorf("insert user role failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace user roles transaction failed: %w", err)
	}

	return nil
}

func (r *PostgresUserRepository) ValidateRoleCodes(ctx context.Context, roleCodes []string) error {
	normalized := normalizeRoleCodes(roleCodes)
	if len(normalized) == 0 {
		return nil
	}

	rows, err := r.db.QueryContext(ctx, `SELECT code FROM app.roles WHERE code = ANY($1);`, pq.Array(normalized))
	if err != nil {
		return fmt.Errorf("validate role codes query failed: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]struct{}, len(normalized))
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return fmt.Errorf("validate role codes scan failed: %w", err)
		}
		existing[code] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("validate role codes rows failed: %w", err)
	}

	for _, code := range normalized {
		if _, ok := existing[code]; !ok {
			return ErrRoleNotFound
		}
	}

	return nil
}

func resolveRoleIDs(ctx context.Context, tx *sql.Tx, roleCodes []string) ([]int64, error) {
	if len(roleCodes) == 0 {
		return nil, nil
	}

	normalized := normalizeRoleCodes(roleCodes)
	if len(normalized) == 0 {
		return nil, nil
	}

	rows, err := tx.QueryContext(ctx, `SELECT id, code FROM app.roles WHERE code = ANY($1);`, pq.Array(normalized))
	if err != nil {
		return nil, fmt.Errorf("resolve role ids query failed: %w", err)
	}
	defer rows.Close()

	byCode := make(map[string]int64, len(normalized))
	for rows.Next() {
		var id int64
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, fmt.Errorf("resolve role ids scan failed: %w", err)
		}
		byCode[code] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolve role ids rows failed: %w", err)
	}

	roleIDs := make([]int64, 0, len(normalized))
	for _, code := range normalized {
		id, ok := byCode[code]
		if !ok {
			return nil, ErrRoleNotFound
		}
		roleIDs = append(roleIDs, id)
	}

	return roleIDs, nil
}

func normalizeRoleCodes(roleCodes []string) []string {
	seen := make(map[string]struct{}, len(roleCodes))
	result := make([]string, 0, len(roleCodes))
	for _, role := range roleCodes {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func nullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}
