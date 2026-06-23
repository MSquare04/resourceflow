package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"resourceflow/backend/internal/model"
)

type ResourceUnavailabilityRepository interface {
	Create(ctx context.Context, params CreateResourceUnavailabilityParams) (model.ResourceUnavailability, error)
	ListByResourceID(ctx context.Context, resourceID int64) ([]model.ResourceUnavailability, error)
	FindByIDAndResourceID(ctx context.Context, resourceID int64, id int64) (model.ResourceUnavailability, error)
	Update(ctx context.Context, resourceID int64, id int64, params UpdateResourceUnavailabilityParams) (model.ResourceUnavailability, error)
	Delete(ctx context.Context, resourceID int64, id int64) (bool, error)
	HasConflict(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error)
}

type PostgresResourceUnavailabilityRepository struct {
	db *sql.DB
}

type CreateResourceUnavailabilityParams struct {
	ResourceID int64
	StartAt    time.Time
	EndAt      time.Time
	Reason     *string
}

type UpdateResourceUnavailabilityParams struct {
	StartAt time.Time
	EndAt   time.Time
	Reason  *string
}

func NewResourceUnavailabilityRepository(db *sql.DB) *PostgresResourceUnavailabilityRepository {
	return &PostgresResourceUnavailabilityRepository{db: db}
}

func (r *PostgresResourceUnavailabilityRepository) Create(
	ctx context.Context,
	params CreateResourceUnavailabilityParams,
) (model.ResourceUnavailability, error) {
	query := `
INSERT INTO app.resource_unavailability (resource_id, start_at, end_at, reason)
VALUES ($1, $2, $3, $4)
RETURNING id, resource_id, start_at, end_at, reason, created_at, updated_at;
`

	row := r.db.QueryRowContext(ctx, query, params.ResourceID, params.StartAt, params.EndAt, nullableString(params.Reason))
	unavailability, err := scanResourceUnavailability(row)
	if err != nil {
		return model.ResourceUnavailability{}, fmt.Errorf("create resource unavailability query failed: %w", err)
	}

	return unavailability, nil
}

func (r *PostgresResourceUnavailabilityRepository) ListByResourceID(
	ctx context.Context,
	resourceID int64,
) ([]model.ResourceUnavailability, error) {
	query := `
SELECT id, resource_id, start_at, end_at, reason, created_at, updated_at
FROM app.resource_unavailability
WHERE resource_id = $1
ORDER BY start_at, id;
`

	rows, err := r.db.QueryContext(ctx, query, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list resource unavailability query failed: %w", err)
	}
	defer rows.Close()

	result := make([]model.ResourceUnavailability, 0)
	for rows.Next() {
		unavailability, err := scanResourceUnavailability(rows)
		if err != nil {
			return nil, fmt.Errorf("list resource unavailability scan failed: %w", err)
		}
		result = append(result, unavailability)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list resource unavailability rows failed: %w", err)
	}

	return result, nil
}

func (r *PostgresResourceUnavailabilityRepository) FindByIDAndResourceID(
	ctx context.Context,
	resourceID int64,
	id int64,
) (model.ResourceUnavailability, error) {
	query := `
SELECT id, resource_id, start_at, end_at, reason, created_at, updated_at
FROM app.resource_unavailability
WHERE resource_id = $1
  AND id = $2
LIMIT 1;
`

	row := r.db.QueryRowContext(ctx, query, resourceID, id)
	unavailability, err := scanResourceUnavailability(row)
	if err != nil {
		return model.ResourceUnavailability{}, fmt.Errorf("find resource unavailability by id query failed: %w", err)
	}

	return unavailability, nil
}

func (r *PostgresResourceUnavailabilityRepository) Update(
	ctx context.Context,
	resourceID int64,
	id int64,
	params UpdateResourceUnavailabilityParams,
) (model.ResourceUnavailability, error) {
	query := `
UPDATE app.resource_unavailability
SET start_at = $3,
    end_at = $4,
    reason = $5,
    updated_at = NOW()
WHERE resource_id = $1
  AND id = $2
RETURNING id, resource_id, start_at, end_at, reason, created_at, updated_at;
`

	row := r.db.QueryRowContext(
		ctx,
		query,
		resourceID,
		id,
		params.StartAt,
		params.EndAt,
		nullableString(params.Reason),
	)
	unavailability, err := scanResourceUnavailability(row)
	if err != nil {
		return model.ResourceUnavailability{}, fmt.Errorf("update resource unavailability query failed: %w", err)
	}

	return unavailability, nil
}

func (r *PostgresResourceUnavailabilityRepository) Delete(ctx context.Context, resourceID int64, id int64) (bool, error) {
	query := `
DELETE FROM app.resource_unavailability
WHERE resource_id = $1
  AND id = $2;
`

	result, err := r.db.ExecContext(ctx, query, resourceID, id)
	if err != nil {
		return false, fmt.Errorf("delete resource unavailability query failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete resource unavailability rows affected failed: %w", err)
	}

	return affected > 0, nil
}

func (r *PostgresResourceUnavailabilityRepository) HasConflict(
	ctx context.Context,
	resourceID int64,
	startAt, endAt time.Time,
) (bool, error) {
	query := `
SELECT EXISTS (
  SELECT 1
  FROM app.resource_unavailability ru
  WHERE ru.resource_id = $1
    AND ru.start_at < $3
    AND ru.end_at > $2
);
`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, resourceID, startAt, endAt).Scan(&exists); err != nil {
		return false, fmt.Errorf("check resource unavailability conflict query failed: %w", err)
	}

	return exists, nil
}

type resourceUnavailabilityScanner interface {
	Scan(dest ...any) error
}

func scanResourceUnavailability(scanner resourceUnavailabilityScanner) (model.ResourceUnavailability, error) {
	var result model.ResourceUnavailability
	var reason sql.NullString

	err := scanner.Scan(
		&result.ID,
		&result.ResourceID,
		&result.StartAt,
		&result.EndAt,
		&reason,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return model.ResourceUnavailability{}, err
	}

	result.Reason = nullableStringPtr(reason)
	return result, nil
}
