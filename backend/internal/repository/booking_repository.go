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
	FindByID(ctx context.Context, id int64) (model.Booking, error)
	CountByUserAndStatuses(ctx context.Context, userID int64, statuses []string) (int64, error)
	HasConflict(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error)
	IsCoveredByAvailability(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error)
	UpdateStatus(ctx context.Context, id int64, params UpdateBookingStatusParams) (model.Booking, error)
	TransitionStatus(ctx context.Context, id int64, expectedFrom []string, params UpdateBookingStatusParams) (model.Booking, string, error)
}

type PostgresBookingRepository struct {
	db *sql.DB
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

func NewBookingRepository(db *sql.DB) *PostgresBookingRepository {
	return &PostgresBookingRepository{db: db}
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

	row := r.db.QueryRowContext(
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

	rows, err := r.db.QueryContext(ctx, query)
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

	rows, err := r.db.QueryContext(ctx, query, userID)
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

	row := r.db.QueryRowContext(ctx, query, id)
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
	if err := r.db.QueryRowContext(ctx, query, userID, pq.Array(statuses)).Scan(&count); err != nil {
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
	if err := r.db.QueryRowContext(ctx, query, resourceID, pq.Array(statuses), startAt, endAt).Scan(&exists); err != nil {
		return false, fmt.Errorf("check booking conflict query failed: %w", err)
	}

	return exists, nil
}

func (r *PostgresBookingRepository) IsCoveredByAvailability(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
	query := `
SELECT EXISTS (
  SELECT 1
  FROM app.resource_availability ra
  WHERE ra.resource_id = $1
    AND ra.start_at <= $2
    AND ra.end_at >= $3
);
`

	var covered bool
	if err := r.db.QueryRowContext(ctx, query, resourceID, startAt, endAt).Scan(&covered); err != nil {
		return false, fmt.Errorf("check booking availability coverage query failed: %w", err)
	}

	return covered, nil
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

	row := r.db.QueryRowContext(
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

	err := r.db.QueryRowContext(
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
