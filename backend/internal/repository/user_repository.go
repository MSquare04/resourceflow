package repository

import (
	"context"
	"database/sql"

	"resourceflow/backend/internal/model"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int64) (model.User, error)
	ListRolesByUserID(ctx context.Context, userID int64) ([]string, error)
}

type PostgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	query := `
SELECT id, full_name, email, password_hash, is_active
FROM app.users
WHERE lower(email) = lower($1)
LIMIT 1;
`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
	)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id int64) (model.User, error) {
	query := `
SELECT id, full_name, email, password_hash, is_active
FROM app.users
WHERE id = $1
LIMIT 1;
`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
	)
	if err != nil {
		return model.User{}, err
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
		return nil, err
	}
	defer rows.Close()

	roles := make([]string, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}
