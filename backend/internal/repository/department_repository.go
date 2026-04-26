package repository

import (
	"context"
	"database/sql"
	"fmt"

	"resourceflow/backend/internal/model"
)

type DepartmentRepository interface {
	Create(ctx context.Context, name, description string, isActive bool) (model.Department, error)
	List(ctx context.Context) ([]model.Department, error)
	FindByID(ctx context.Context, id int64) (model.Department, error)
	Update(ctx context.Context, id int64, name, description string, isActive bool) (model.Department, error)
}

type PostgresDepartmentRepository struct {
	db *sql.DB
}

func NewDepartmentRepository(db *sql.DB) *PostgresDepartmentRepository {
	return &PostgresDepartmentRepository{db: db}
}

func (r *PostgresDepartmentRepository) Create(ctx context.Context, name, description string, isActive bool) (model.Department, error) {
	query := `
INSERT INTO app.departments (name, description, is_active)
VALUES ($1, $2, $3)
RETURNING id, name, COALESCE(description, ''), is_active;
`

	var department model.Department
	err := r.db.QueryRowContext(ctx, query, name, description, isActive).Scan(
		&department.ID,
		&department.Name,
		&department.Description,
		&department.IsActive,
	)
	if err != nil {
		return model.Department{}, fmt.Errorf("create department query failed: %w", err)
	}

	return department, nil
}

func (r *PostgresDepartmentRepository) List(ctx context.Context) ([]model.Department, error) {
	query := `
SELECT id, name, COALESCE(description, ''), is_active
FROM app.departments
ORDER BY id;
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list departments query failed: %w", err)
	}
	defer rows.Close()

	departments := make([]model.Department, 0)
	for rows.Next() {
		var department model.Department
		if err := rows.Scan(&department.ID, &department.Name, &department.Description, &department.IsActive); err != nil {
			return nil, fmt.Errorf("list departments scan failed: %w", err)
		}
		departments = append(departments, department)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list departments rows failed: %w", err)
	}

	return departments, nil
}

func (r *PostgresDepartmentRepository) FindByID(ctx context.Context, id int64) (model.Department, error) {
	query := `
SELECT id, name, COALESCE(description, ''), is_active
FROM app.departments
WHERE id = $1
LIMIT 1;
`

	var department model.Department
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&department.ID,
		&department.Name,
		&department.Description,
		&department.IsActive,
	)
	if err != nil {
		return model.Department{}, fmt.Errorf("find department by id query failed: %w", err)
	}

	return department, nil
}

func (r *PostgresDepartmentRepository) Update(ctx context.Context, id int64, name, description string, isActive bool) (model.Department, error) {
	query := `
UPDATE app.departments
SET name = $2,
    description = $3,
    is_active = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, COALESCE(description, ''), is_active;
`

	var department model.Department
	err := r.db.QueryRowContext(ctx, query, id, name, description, isActive).Scan(
		&department.ID,
		&department.Name,
		&department.Description,
		&department.IsActive,
	)
	if err != nil {
		return model.Department{}, fmt.Errorf("update department query failed: %w", err)
	}

	return department, nil
}
