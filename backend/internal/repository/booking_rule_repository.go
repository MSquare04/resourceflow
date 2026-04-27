package repository

import (
	"context"
	"database/sql"
	"fmt"

	"resourceflow/backend/internal/model"
)

type BookingRuleRepository interface {
	Create(ctx context.Context, params CreateBookingRuleParams) (model.BookingRule, error)
	List(ctx context.Context) ([]model.BookingRule, error)
	FindByID(ctx context.Context, id int64) (model.BookingRule, error)
	Update(ctx context.Context, id int64, params UpdateBookingRuleParams) (model.BookingRule, error)
}

type PostgresBookingRuleRepository struct {
	db *sql.DB
}

type CreateBookingRuleParams struct {
	ResourceTypeID           int64
	MinDurationMinutes       int32
	MaxDurationMinutes       int32
	MaxActiveBookingsPerUser int32
	RequiresApproval         bool
	BookingHorizonDays       int32
	IsActive                 bool
}

type UpdateBookingRuleParams struct {
	ResourceTypeID           int64
	MinDurationMinutes       int32
	MaxDurationMinutes       int32
	MaxActiveBookingsPerUser int32
	RequiresApproval         bool
	BookingHorizonDays       int32
	IsActive                 bool
}

func NewBookingRuleRepository(db *sql.DB) *PostgresBookingRuleRepository {
	return &PostgresBookingRuleRepository{db: db}
}

func (r *PostgresBookingRuleRepository) Create(ctx context.Context, params CreateBookingRuleParams) (model.BookingRule, error) {
	query := `
INSERT INTO app.booking_rules (
  resource_type_id,
  min_duration_minutes,
  max_duration_minutes,
  max_active_bookings_per_user,
  requires_approval,
  booking_horizon_days,
  is_active
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING
  id,
  resource_type_id,
  min_duration_minutes,
  max_duration_minutes,
  max_active_bookings_per_user,
  requires_approval,
  booking_horizon_days,
  is_active,
  created_at,
  updated_at;
`

	var rule model.BookingRule
	err := r.db.QueryRowContext(
		ctx,
		query,
		params.ResourceTypeID,
		params.MinDurationMinutes,
		params.MaxDurationMinutes,
		params.MaxActiveBookingsPerUser,
		params.RequiresApproval,
		params.BookingHorizonDays,
		params.IsActive,
	).Scan(
		&rule.ID,
		&rule.ResourceTypeID,
		&rule.MinDurationMinutes,
		&rule.MaxDurationMinutes,
		&rule.MaxActiveBookingsPerUser,
		&rule.RequiresApproval,
		&rule.BookingHorizonDays,
		&rule.IsActive,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		return model.BookingRule{}, fmt.Errorf("create booking rule query failed: %w", err)
	}

	return rule, nil
}

func (r *PostgresBookingRuleRepository) List(ctx context.Context) ([]model.BookingRule, error) {
	query := `
SELECT
  id,
  resource_type_id,
  min_duration_minutes,
  max_duration_minutes,
  max_active_bookings_per_user,
  requires_approval,
  booking_horizon_days,
  is_active,
  created_at,
  updated_at
FROM app.booking_rules
ORDER BY id;
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list booking rules query failed: %w", err)
	}
	defer rows.Close()

	result := make([]model.BookingRule, 0)
	for rows.Next() {
		var rule model.BookingRule
		if err := rows.Scan(
			&rule.ID,
			&rule.ResourceTypeID,
			&rule.MinDurationMinutes,
			&rule.MaxDurationMinutes,
			&rule.MaxActiveBookingsPerUser,
			&rule.RequiresApproval,
			&rule.BookingHorizonDays,
			&rule.IsActive,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list booking rules scan failed: %w", err)
		}
		result = append(result, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list booking rules rows failed: %w", err)
	}

	return result, nil
}

func (r *PostgresBookingRuleRepository) FindByID(ctx context.Context, id int64) (model.BookingRule, error) {
	query := `
SELECT
  id,
  resource_type_id,
  min_duration_minutes,
  max_duration_minutes,
  max_active_bookings_per_user,
  requires_approval,
  booking_horizon_days,
  is_active,
  created_at,
  updated_at
FROM app.booking_rules
WHERE id = $1
LIMIT 1;
`

	var rule model.BookingRule
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID,
		&rule.ResourceTypeID,
		&rule.MinDurationMinutes,
		&rule.MaxDurationMinutes,
		&rule.MaxActiveBookingsPerUser,
		&rule.RequiresApproval,
		&rule.BookingHorizonDays,
		&rule.IsActive,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		return model.BookingRule{}, fmt.Errorf("find booking rule by id query failed: %w", err)
	}

	return rule, nil
}

func (r *PostgresBookingRuleRepository) Update(ctx context.Context, id int64, params UpdateBookingRuleParams) (model.BookingRule, error) {
	query := `
UPDATE app.booking_rules
SET resource_type_id = $2,
    min_duration_minutes = $3,
    max_duration_minutes = $4,
    max_active_bookings_per_user = $5,
    requires_approval = $6,
    booking_horizon_days = $7,
    is_active = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING
  id,
  resource_type_id,
  min_duration_minutes,
  max_duration_minutes,
  max_active_bookings_per_user,
  requires_approval,
  booking_horizon_days,
  is_active,
  created_at,
  updated_at;
`

	var rule model.BookingRule
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
		params.ResourceTypeID,
		params.MinDurationMinutes,
		params.MaxDurationMinutes,
		params.MaxActiveBookingsPerUser,
		params.RequiresApproval,
		params.BookingHorizonDays,
		params.IsActive,
	).Scan(
		&rule.ID,
		&rule.ResourceTypeID,
		&rule.MinDurationMinutes,
		&rule.MaxDurationMinutes,
		&rule.MaxActiveBookingsPerUser,
		&rule.RequiresApproval,
		&rule.BookingHorizonDays,
		&rule.IsActive,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		return model.BookingRule{}, fmt.Errorf("update booking rule query failed: %w", err)
	}

	return rule, nil
}
