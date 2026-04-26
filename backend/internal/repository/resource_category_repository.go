package repository

import (
	"context"
	"database/sql"
	"fmt"

	"resourceflow/backend/internal/model"
)

type ResourceCategoryRepository interface {
	Create(ctx context.Context, code, name, description string, isActive bool) (model.ResourceCategory, error)
	List(ctx context.Context) ([]model.ResourceCategory, error)
	FindByID(ctx context.Context, id int64) (model.ResourceCategory, error)
	Update(ctx context.Context, id int64, code, name, description string, isActive bool) (model.ResourceCategory, error)
}

type PostgresResourceCategoryRepository struct {
	db *sql.DB
}

func NewResourceCategoryRepository(db *sql.DB) *PostgresResourceCategoryRepository {
	return &PostgresResourceCategoryRepository{db: db}
}

func (r *PostgresResourceCategoryRepository) Create(ctx context.Context, code, name, description string, isActive bool) (model.ResourceCategory, error) {
	query := `
INSERT INTO app.resource_categories (code, name, description, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, code, name, COALESCE(description, ''), is_active;
`

	var category model.ResourceCategory
	err := r.db.QueryRowContext(ctx, query, code, name, description, isActive).Scan(
		&category.ID,
		&category.Code,
		&category.Name,
		&category.Description,
		&category.IsActive,
	)
	if err != nil {
		return model.ResourceCategory{}, fmt.Errorf("create resource category query failed: %w", err)
	}

	return category, nil
}

func (r *PostgresResourceCategoryRepository) List(ctx context.Context) ([]model.ResourceCategory, error) {
	query := `
SELECT id, code, name, COALESCE(description, ''), is_active
FROM app.resource_categories
ORDER BY id;
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list resource categories query failed: %w", err)
	}
	defer rows.Close()

	categories := make([]model.ResourceCategory, 0)
	for rows.Next() {
		var category model.ResourceCategory
		if err := rows.Scan(&category.ID, &category.Code, &category.Name, &category.Description, &category.IsActive); err != nil {
			return nil, fmt.Errorf("list resource categories scan failed: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list resource categories rows failed: %w", err)
	}

	return categories, nil
}

func (r *PostgresResourceCategoryRepository) FindByID(ctx context.Context, id int64) (model.ResourceCategory, error) {
	query := `
SELECT id, code, name, COALESCE(description, ''), is_active
FROM app.resource_categories
WHERE id = $1
LIMIT 1;
`

	var category model.ResourceCategory
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Code,
		&category.Name,
		&category.Description,
		&category.IsActive,
	)
	if err != nil {
		return model.ResourceCategory{}, fmt.Errorf("find resource category by id query failed: %w", err)
	}

	return category, nil
}

func (r *PostgresResourceCategoryRepository) Update(ctx context.Context, id int64, code, name, description string, isActive bool) (model.ResourceCategory, error) {
	query := `
UPDATE app.resource_categories
SET code = $2,
    name = $3,
    description = $4,
    is_active = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING id, code, name, COALESCE(description, ''), is_active;
`

	var category model.ResourceCategory
	err := r.db.QueryRowContext(ctx, query, id, code, name, description, isActive).Scan(
		&category.ID,
		&category.Code,
		&category.Name,
		&category.Description,
		&category.IsActive,
	)
	if err != nil {
		return model.ResourceCategory{}, fmt.Errorf("update resource category query failed: %w", err)
	}

	return category, nil
}
