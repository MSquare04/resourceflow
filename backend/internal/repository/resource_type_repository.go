package repository

import (
	"context"
	"database/sql"

	"resourceflow/backend/internal/model"
)

type ResourceTypeRepository interface {
	Create(ctx context.Context, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error)
	List(ctx context.Context) ([]model.ResourceType, error)
	FindByID(ctx context.Context, id int64) (model.ResourceType, error)
	Update(ctx context.Context, id int64, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error)
}

type PostgresResourceTypeRepository struct {
	db *sql.DB
}

func NewResourceTypeRepository(db *sql.DB) *PostgresResourceTypeRepository {
	return &PostgresResourceTypeRepository{db: db}
}

func (r *PostgresResourceTypeRepository) Create(ctx context.Context, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error) {
	query := `
INSERT INTO app.resource_types (category_id, code, name, description, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, category_id, code, name, COALESCE(description, ''), is_active;
`

	var resourceType model.ResourceType
	err := r.db.QueryRowContext(ctx, query, categoryID, code, name, description, isActive).Scan(
		&resourceType.ID,
		&resourceType.CategoryID,
		&resourceType.Code,
		&resourceType.Name,
		&resourceType.Description,
		&resourceType.IsActive,
	)
	if err != nil {
		return model.ResourceType{}, err
	}

	return resourceType, nil
}

func (r *PostgresResourceTypeRepository) List(ctx context.Context) ([]model.ResourceType, error) {
	query := `
SELECT id, category_id, code, name, COALESCE(description, ''), is_active
FROM app.resource_types
ORDER BY id;
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resourceTypes := make([]model.ResourceType, 0)
	for rows.Next() {
		var resourceType model.ResourceType
		if err := rows.Scan(
			&resourceType.ID,
			&resourceType.CategoryID,
			&resourceType.Code,
			&resourceType.Name,
			&resourceType.Description,
			&resourceType.IsActive,
		); err != nil {
			return nil, err
		}
		resourceTypes = append(resourceTypes, resourceType)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resourceTypes, nil
}

func (r *PostgresResourceTypeRepository) FindByID(ctx context.Context, id int64) (model.ResourceType, error) {
	query := `
SELECT id, category_id, code, name, COALESCE(description, ''), is_active
FROM app.resource_types
WHERE id = $1
LIMIT 1;
`

	var resourceType model.ResourceType
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&resourceType.ID,
		&resourceType.CategoryID,
		&resourceType.Code,
		&resourceType.Name,
		&resourceType.Description,
		&resourceType.IsActive,
	)
	if err != nil {
		return model.ResourceType{}, err
	}

	return resourceType, nil
}

func (r *PostgresResourceTypeRepository) Update(ctx context.Context, id int64, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error) {
	query := `
UPDATE app.resource_types
SET category_id = $2,
    code = $3,
    name = $4,
    description = $5,
    is_active = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING id, category_id, code, name, COALESCE(description, ''), is_active;
`

	var resourceType model.ResourceType
	err := r.db.QueryRowContext(ctx, query, id, categoryID, code, name, description, isActive).Scan(
		&resourceType.ID,
		&resourceType.CategoryID,
		&resourceType.Code,
		&resourceType.Name,
		&resourceType.Description,
		&resourceType.IsActive,
	)
	if err != nil {
		return model.ResourceType{}, err
	}

	return resourceType, nil
}
