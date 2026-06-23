package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"resourceflow/backend/internal/model"
)

type BookingRepository interface {
	Create(ctx context.Context, params CreateBookingParams) (model.Booking, error)
	List(ctx context.Context) ([]model.Booking, error)
	ListByUserID(ctx context.Context, userID int64) ([]model.Booking, error)
	ListBusyIntervalsByResourceID(ctx context.Context, resourceID int64, statuses []string, from, until time.Time) ([]model.Booking, error)
	FindByID(ctx context.Context, id int64) (model.Booking, error)
	CountByUserAndStatuses(ctx context.Context, userID int64, statuses []string) (int64, error)
	HasConflict(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error)
	ProcessExpired(ctx context.Context, now time.Time) (ExpiredBookingProcessingResult, error)
	UpdateStatus(ctx context.Context, id int64, params UpdateBookingStatusParams) (model.Booking, error)
	TransitionStatus(ctx context.Context, id int64, expectedFrom []string, params UpdateBookingStatusParams) (model.Booking, string, error)
}

type BookingRepositoryTxRunner interface {
	BookingRepository
	WithTransaction(ctx context.Context, fn func(repo BookingRepository) error) error
}

type bookingQueryRunner interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type PostgresBookingRepository struct {
	db     *sql.DB
	runner bookingQueryRunner
}

type CreateBookingParams struct {
	ResourceID int64
	UserID     int64
	StartAt    time.Time
	EndAt      time.Time
	Purpose    *string
	Status     string
}

type UpdateBookingStatusParams struct {
	Status           string
	ApprovedByUserID *int64
	ApprovedAt       *time.Time
	CancelledAt      *time.Time
	CompletedAt      *time.Time
}

type ExpiredBookingProcessingResult struct {
	CompletedCount int64
	CancelledCount int64
}

func NewBookingRepository(db *sql.DB) *PostgresBookingRepository {
	return &PostgresBookingRepository{db: db, runner: db}
}

func (r *PostgresBookingRepository) WithTransaction(ctx context.Context, fn func(repo BookingRepository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin booking transaction failed: %w", err)
	}

	txRepo := &PostgresBookingRepository{
		db:     r.db,
		runner: tx,
	}
	if err := fn(txRepo); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("rollback booking transaction failed: %v: %w", rollbackErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit booking transaction failed: %w", err)
	}

	return nil
}

func (r *PostgresBookingRepository) Create(ctx context.Context, params CreateBookingParams) (model.Booking, error) {
	query := `
INSERT INTO app.bookings (resource_id, user_id, start_at, end_at, purpose, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING
  id,
  resource_id,
  user_id,
  start_at,
  end_at,
  purpose,
  status,
  approved_by_user_id,
  approved_at,
  cancelled_at,
  completed_at,
  created_at,
  updated_at;
`

	row := r.runner.QueryRowContext(
		ctx,
		query,
		params.ResourceID,
		params.UserID,
		params.StartAt,
		params.EndAt,
		nullableString(params.Purpose),
		params.Status,
	)
	booking, err := scanBooking(row)
	if err != nil {
		return model.Booking{}, fmt.Errorf("create booking query failed: %w", err)
	}

	return booking, nil
}

func (r *PostgresBookingRepository) List(ctx context.Context) ([]model.Booking, error) {
	query := `
SELECT
  id,
  resource_id,
  user_id,
  start_at,
  end_at,
  purpose,
  status,
  approved_by_user_id,
  approved_at,
  cancelled_at,
  completed_at,
  created_at,
  updated_at
FROM app.bookings
ORDER BY id DESC;
`

	rows, err := r.runner.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list bookings query failed: %w", err)
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("list bookings scan failed: %w", err)
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bookings rows failed: %w", err)
	}

	return bookings, nil
}

func (r *PostgresBookingRepository) ListByUserID(ctx context.Context, userID int64) ([]model.Booking, error) {
	query := `
SELECT
  id,
  resource_id,
  user_id,
  start_at,
  end_at,
  purpose,
  status,
  approved_by_user_id,
  approved_at,
  cancelled_at,
  completed_at,
  created_at,
  updated_at
FROM app.bookings
WHERE user_id = $1
ORDER BY id DESC;
`

	rows, err := r.runner.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list bookings by user query failed: %w", err)
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("list bookings by user scan failed: %w", err)
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bookings by user rows failed: %w", err)
	}

	return bookings, nil
}

func (r *PostgresBookingRepository) ListBusyIntervalsByResourceID(
	ctx context.Context,
	resourceID int64,
	statuses []string,
	from, until time.Time,
) ([]model.Booking, error) {
	query := `
SELECT
  id,
  resource_id,
  user_id,
  start_at,
  end_at,
  purpose,
  status,
  approved_by_user_id,
  approved_at,
  cancelled_at,
  completed_at,
  created_at,
  updated_at
FROM app.bookings
WHERE resource_id = $1
  AND status = ANY($2)
  AND end_at > $3
  AND start_at < $4
ORDER BY start_at ASC, end_at ASC;
`

	rows, err := r.runner.QueryContext(ctx, query, resourceID, pq.Array(statuses), from, until)
	if err != nil {
		return nil, fmt.Errorf("list busy intervals by resource query failed: %w", err)
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("list busy intervals by resource scan failed: %w", err)
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list busy intervals by resource rows failed: %w", err)
	}

	return bookings, nil
}

func (r *PostgresBookingRepository) FindByID(ctx context.Context, id int64) (model.Booking, error) {
	query := `
SELECT
  id,
  resource_id,
  user_id,
  start_at,
  end_at,
  purpose,
  status,
  approved_by_user_id,
  approved_at,
  cancelled_at,
  completed_at,
  created_at,
  updated_at
FROM app.bookings
WHERE id = $1
LIMIT 1;
`

	row := r.runner.QueryRowContext(ctx, query, id)
	booking, err := scanBooking(row)
	if err != nil {
		return model.Booking{}, fmt.Errorf("find booking by id query failed: %w", err)
	}

	return booking, nil
}

func (r *PostgresBookingRepository) CountByUserAndStatuses(ctx context.Context, userID int64, statuses []string) (int64, error) {
	query := `
SELECT COUNT(1)
FROM app.bookings
WHERE user_id = $1
  AND status = ANY($2);
`

	var count int64
	if err := r.runner.QueryRowContext(ctx, query, userID, pq.Array(statuses)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count bookings by user and statuses query failed: %w", err)
	}

	return count, nil
}

func (r *PostgresBookingRepository) HasConflict(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
	query := `
SELECT EXISTS (
  SELECT 1
  FROM app.bookings b
  WHERE b.resource_id = $1
    AND b.status = ANY($2)
    AND b.start_at < $4
    AND b.end_at > $3
);
`

	var exists bool
	if err := r.runner.QueryRowContext(ctx, query, resourceID, pq.Array(statuses), startAt, endAt).Scan(&exists); err != nil {
		return false, fmt.Errorf("check booking conflict query failed: %w", err)
	}

	return exists, nil
}

func (r *PostgresBookingRepository) ProcessExpired(ctx context.Context, now time.Time) (ExpiredBookingProcessingResult, error) {
	query := `
WITH completed AS (
  UPDATE app.bookings
  SET status = 'completed',
      completed_at = COALESCE(completed_at, $1),
      updated_at = $1
  WHERE status = 'confirmed'
    AND end_at <= $1
  RETURNING id
),
cancelled AS (
  UPDATE app.bookings
  SET status = 'cancelled',
      cancelled_at = COALESCE(cancelled_at, $1),
      updated_at = $1
  WHERE status = 'pending'
    AND end_at <= $1
  RETURNING id
)
SELECT
  (SELECT COUNT(*) FROM completed),
  (SELECT COUNT(*) FROM cancelled);
`

	var result ExpiredBookingProcessingResult
	if err := r.runner.QueryRowContext(ctx, query, now.UTC()).Scan(&result.CompletedCount, &result.CancelledCount); err != nil {
		return ExpiredBookingProcessingResult{}, fmt.Errorf("process expired bookings query failed: %w", err)
	}

	return result, nil
}

func (r *PostgresBookingRepository) UpdateStatus(ctx context.Context, id int64, params UpdateBookingStatusParams) (model.Booking, error) {
	query := `
UPDATE app.bookings
SET status = $2,
    approved_by_user_id = $3,
    approved_at = $4,
    cancelled_at = $5,
    completed_at = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING
  id,
  resource_id,
  user_id,
  start_at,
  end_at,
  purpose,
  status,
  approved_by_user_id,
  approved_at,
  cancelled_at,
  completed_at,
  created_at,
  updated_at;
`

	row := r.runner.QueryRowContext(
		ctx,
		query,
		id,
		params.Status,
		nullableInt64(params.ApprovedByUserID),
		nullableTime(params.ApprovedAt),
		nullableTime(params.CancelledAt),
		nullableTime(params.CompletedAt),
	)
	booking, err := scanBooking(row)
	if err != nil {
		return model.Booking{}, fmt.Errorf("update booking status query failed: %w", err)
	}

	return booking, nil
}

func (r *PostgresBookingRepository) TransitionStatus(
	ctx context.Context,
	id int64,
	expectedFrom []string,
	params UpdateBookingStatusParams,
) (model.Booking, string, error) {
	query := `
WITH current AS (
  SELECT b.id, b.status
  FROM app.bookings b
  WHERE b.id = $1
    AND b.status = ANY($2)
  FOR UPDATE
),
updated AS (
  UPDATE app.bookings b
  SET status = $3,
      approved_by_user_id = COALESCE($4, b.approved_by_user_id),
      approved_at = COALESCE($5, b.approved_at),
      cancelled_at = $6,
      completed_at = $7,
      updated_at = NOW()
  FROM current c
  WHERE b.id = c.id
  RETURNING
    b.id,
    b.resource_id,
    b.user_id,
    b.start_at,
    b.end_at,
    b.purpose,
    b.status,
    b.approved_by_user_id,
    b.approved_at,
    b.cancelled_at,
    b.completed_at,
    b.created_at,
    b.updated_at,
    c.status AS status_from
)
SELECT
  id,
  resource_id,
  user_id,
  start_at,
  end_at,
  purpose,
  status,
  approved_by_user_id,
  approved_at,
  cancelled_at,
  completed_at,
  created_at,
  updated_at,
  status_from
FROM updated;
`

	var booking model.Booking
	var statusFrom string
	var purpose sql.NullString
	var approvedByUserID sql.NullInt64
	var approvedAt sql.NullTime
	var cancelledAt sql.NullTime
	var completedAt sql.NullTime

	err := r.runner.QueryRowContext(
		ctx,
		query,
		id,
		pq.Array(expectedFrom),
		params.Status,
		nullableInt64(params.ApprovedByUserID),
		nullableTime(params.ApprovedAt),
		nullableTime(params.CancelledAt),
		nullableTime(params.CompletedAt),
	).Scan(
		&booking.ID,
		&booking.ResourceID,
		&booking.UserID,
		&booking.StartAt,
		&booking.EndAt,
		&purpose,
		&booking.Status,
		&approvedByUserID,
		&approvedAt,
		&cancelledAt,
		&completedAt,
		&booking.CreatedAt,
		&booking.UpdatedAt,
		&statusFrom,
	)
	if err != nil {
		return model.Booking{}, "", fmt.Errorf("transition booking status query failed: %w", err)
	}

	booking.Purpose = nullableStringPtr(purpose)
	booking.ApprovedByUserID = nullableInt64Ptr(approvedByUserID)
	booking.ApprovedAt = nullableTimePtr(approvedAt)
	booking.CancelledAt = nullableTimePtr(cancelledAt)
	booking.CompletedAt = nullableTimePtr(completedAt)

	return booking, statusFrom, nil
}

type bookingScanner interface {
	Scan(dest ...any) error
}

func scanBooking(scanner bookingScanner) (model.Booking, error) {
	var booking model.Booking
	var purpose sql.NullString
	var approvedByUserID sql.NullInt64
	var approvedAt sql.NullTime
	var cancelledAt sql.NullTime
	var completedAt sql.NullTime

	err := scanner.Scan(
		&booking.ID,
		&booking.ResourceID,
		&booking.UserID,
		&booking.StartAt,
		&booking.EndAt,
		&purpose,
		&booking.Status,
		&approvedByUserID,
		&approvedAt,
		&cancelledAt,
		&completedAt,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)
	if err != nil {
		return model.Booking{}, err
	}

	booking.Purpose = nullableStringPtr(purpose)
	booking.ApprovedByUserID = nullableInt64Ptr(approvedByUserID)
	booking.ApprovedAt = nullableTimePtr(approvedAt)
	booking.CancelledAt = nullableTimePtr(cancelledAt)
	booking.CompletedAt = nullableTimePtr(completedAt)

	return booking, nil
}

func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
