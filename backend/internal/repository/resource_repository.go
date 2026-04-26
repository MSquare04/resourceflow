package repository

import (
	"context"
	"database/sql"

	"resourceflow/backend/internal/model"
)

type ResourceRepository interface {
	Create(ctx context.Context, params CreateResourceParams) (model.Resource, error)
	List(ctx context.Context) ([]model.Resource, error)
	FindByID(ctx context.Context, id int64) (model.Resource, error)
	Update(ctx context.Context, id int64, params UpdateResourceParams) (model.Resource, error)
}

type PostgresResourceRepository struct {
	db *sql.DB
}

type CreateResourceParams struct {
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

type UpdateResourceParams struct {
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

func NewResourceRepository(db *sql.DB) *PostgresResourceRepository {
	return &PostgresResourceRepository{db: db}
}

func (r *PostgresResourceRepository) Create(ctx context.Context, params CreateResourceParams) (model.Resource, error) {
	query := `
INSERT INTO app.resources (name, description, category_id, type_id, department_id, location, capacity, is_bookable, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, name, COALESCE(description, ''), category_id, type_id, department_id, location, capacity, is_bookable, is_active;
`

	var resource model.Resource
	var departmentID sql.NullInt64
	var location sql.NullString
	var capacity sql.NullInt64
	err := r.db.QueryRowContext(
		ctx,
		query,
		params.Name,
		params.Description,
		params.CategoryID,
		params.TypeID,
		nullableInt64(params.DepartmentID),
		nullableString(params.Location),
		nullableInt64(params.Capacity),
		params.IsBookable,
		params.IsActive,
	).Scan(
		&resource.ID,
		&resource.Name,
		&resource.Description,
		&resource.CategoryID,
		&resource.TypeID,
		&departmentID,
		&location,
		&capacity,
		&resource.IsBookable,
		&resource.IsActive,
	)
	if err != nil {
		return model.Resource{}, err
	}

	resource.DepartmentID = nullableInt64Ptr(departmentID)
	resource.Location = nullableStringPtr(location)
	resource.Capacity = nullableInt64Ptr(capacity)

	return resource, nil
}

func (r *PostgresResourceRepository) List(ctx context.Context) ([]model.Resource, error) {
	query := `
SELECT id, name, COALESCE(description, ''), category_id, type_id, department_id, location, capacity, is_bookable, is_active
FROM app.resources
ORDER BY id;
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := make([]model.Resource, 0)
	for rows.Next() {
		var resource model.Resource
		var departmentID sql.NullInt64
		var location sql.NullString
		var capacity sql.NullInt64
		if err := rows.Scan(
			&resource.ID,
			&resource.Name,
			&resource.Description,
			&resource.CategoryID,
			&resource.TypeID,
			&departmentID,
			&location,
			&capacity,
			&resource.IsBookable,
			&resource.IsActive,
		); err != nil {
			return nil, err
		}
		resource.DepartmentID = nullableInt64Ptr(departmentID)
		resource.Location = nullableStringPtr(location)
		resource.Capacity = nullableInt64Ptr(capacity)
		resources = append(resources, resource)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resources, nil
}

func (r *PostgresResourceRepository) FindByID(ctx context.Context, id int64) (model.Resource, error) {
	query := `
SELECT id, name, COALESCE(description, ''), category_id, type_id, department_id, location, capacity, is_bookable, is_active
FROM app.resources
WHERE id = $1
LIMIT 1;
`

	var resource model.Resource
	var departmentID sql.NullInt64
	var location sql.NullString
	var capacity sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&resource.ID,
		&resource.Name,
		&resource.Description,
		&resource.CategoryID,
		&resource.TypeID,
		&departmentID,
		&location,
		&capacity,
		&resource.IsBookable,
		&resource.IsActive,
	)
	if err != nil {
		return model.Resource{}, err
	}

	resource.DepartmentID = nullableInt64Ptr(departmentID)
	resource.Location = nullableStringPtr(location)
	resource.Capacity = nullableInt64Ptr(capacity)

	return resource, nil
}

func (r *PostgresResourceRepository) Update(ctx context.Context, id int64, params UpdateResourceParams) (model.Resource, error) {
	query := `
UPDATE app.resources
SET name = $2,
    description = $3,
    category_id = $4,
    type_id = $5,
    department_id = $6,
    location = $7,
    capacity = $8,
    is_bookable = $9,
    is_active = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, COALESCE(description, ''), category_id, type_id, department_id, location, capacity, is_bookable, is_active;
`

	var resource model.Resource
	var departmentID sql.NullInt64
	var location sql.NullString
	var capacity sql.NullInt64
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
		params.Name,
		params.Description,
		params.CategoryID,
		params.TypeID,
		nullableInt64(params.DepartmentID),
		nullableString(params.Location),
		nullableInt64(params.Capacity),
		params.IsBookable,
		params.IsActive,
	).Scan(
		&resource.ID,
		&resource.Name,
		&resource.Description,
		&resource.CategoryID,
		&resource.TypeID,
		&departmentID,
		&location,
		&capacity,
		&resource.IsBookable,
		&resource.IsActive,
	)
	if err != nil {
		return model.Resource{}, err
	}

	resource.DepartmentID = nullableInt64Ptr(departmentID)
	resource.Location = nullableStringPtr(location)
	resource.Capacity = nullableInt64Ptr(capacity)

	return resource, nil
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func nullableStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}
