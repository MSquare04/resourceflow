package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"resourceflow/backend/internal/model"
)

type ResourceAvailabilityRepository interface {
	Create(ctx context.Context, params CreateResourceAvailabilityParams) (model.ResourceAvailability, error)
	ListByResourceID(ctx context.Context, resourceID int64) ([]model.ResourceAvailability, error)
	FindByIDAndResourceID(ctx context.Context, resourceID int64, id int64) (model.ResourceAvailability, error)
	Update(ctx context.Context, resourceID int64, id int64, params UpdateResourceAvailabilityParams) (model.ResourceAvailability, error)
	Delete(ctx context.Context, resourceID int64, id int64) (bool, error)
}

type PostgresResourceAvailabilityRepository struct {
	db *sql.DB
}

type CreateResourceAvailabilityParams struct {
	ResourceID int64
	StartAt    time.Time
	EndAt      time.Time
}

type UpdateResourceAvailabilityParams struct {
	StartAt time.Time
	EndAt   time.Time
}

func NewResourceAvailabilityRepository(db *sql.DB) *PostgresResourceAvailabilityRepository {
	return &PostgresResourceAvailabilityRepository{db: db}
}

func (r *PostgresResourceAvailabilityRepository) Create(ctx context.Context, params CreateResourceAvailabilityParams) (model.ResourceAvailability, error) {
	query := `
INSERT INTO app.resource_availability (resource_id, start_at, end_at)
VALUES ($1, $2, $3)
RETURNING id, resource_id, start_at, end_at, created_at, updated_at;
`

	var availability model.ResourceAvailability
	err := r.db.QueryRowContext(ctx, query, params.ResourceID, params.StartAt, params.EndAt).Scan(
		&availability.ID,
		&availability.ResourceID,
		&availability.StartAt,
		&availability.EndAt,
		&availability.CreatedAt,
		&availability.UpdatedAt,
	)
	if err != nil {
		return model.ResourceAvailability{}, fmt.Errorf("create resource availability query failed: %w", err)
	}

	return availability, nil
}

func (r *PostgresResourceAvailabilityRepository) ListByResourceID(ctx context.Context, resourceID int64) ([]model.ResourceAvailability, error) {
	query := `
SELECT id, resource_id, start_at, end_at, created_at, updated_at
FROM app.resource_availability
WHERE resource_id = $1
ORDER BY start_at, id;
`

	rows, err := r.db.QueryContext(ctx, query, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list resource availability query failed: %w", err)
	}
	defer rows.Close()

	result := make([]model.ResourceAvailability, 0)
	for rows.Next() {
		var availability model.ResourceAvailability
		if err := rows.Scan(
			&availability.ID,
			&availability.ResourceID,
			&availability.StartAt,
			&availability.EndAt,
			&availability.CreatedAt,
			&availability.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list resource availability scan failed: %w", err)
		}
		result = append(result, availability)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list resource availability rows failed: %w", err)
	}

	return result, nil
}

func (r *PostgresResourceAvailabilityRepository) FindByIDAndResourceID(ctx context.Context, resourceID int64, id int64) (model.ResourceAvailability, error) {
	query := `
SELECT id, resource_id, start_at, end_at, created_at, updated_at
FROM app.resource_availability
WHERE resource_id = $1
  AND id = $2
LIMIT 1;
`

	var availability model.ResourceAvailability
	err := r.db.QueryRowContext(ctx, query, resourceID, id).Scan(
		&availability.ID,
		&availability.ResourceID,
		&availability.StartAt,
		&availability.EndAt,
		&availability.CreatedAt,
		&availability.UpdatedAt,
	)
	if err != nil {
		return model.ResourceAvailability{}, fmt.Errorf("find resource availability by id query failed: %w", err)
	}

	return availability, nil
}

func (r *PostgresResourceAvailabilityRepository) Update(ctx context.Context, resourceID int64, id int64, params UpdateResourceAvailabilityParams) (model.ResourceAvailability, error) {
	query := `
UPDATE app.resource_availability
SET start_at = $3,
    end_at = $4,
    updated_at = NOW()
WHERE resource_id = $1
  AND id = $2
RETURNING id, resource_id, start_at, end_at, created_at, updated_at;
`

	var availability model.ResourceAvailability
	err := r.db.QueryRowContext(ctx, query, resourceID, id, params.StartAt, params.EndAt).Scan(
		&availability.ID,
		&availability.ResourceID,
		&availability.StartAt,
		&availability.EndAt,
		&availability.CreatedAt,
		&availability.UpdatedAt,
	)
	if err != nil {
		return model.ResourceAvailability{}, fmt.Errorf("update resource availability query failed: %w", err)
	}

	return availability, nil
}

func (r *PostgresResourceAvailabilityRepository) Delete(ctx context.Context, resourceID int64, id int64) (bool, error) {
	query := `
DELETE FROM app.resource_availability
WHERE resource_id = $1
  AND id = $2;
`

	result, err := r.db.ExecContext(ctx, query, resourceID, id)
	if err != nil {
		return false, fmt.Errorf("delete resource availability query failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete resource availability rows affected failed: %w", err)
	}

	return affected > 0, nil
}
