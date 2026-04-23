package db

import (
	"database/sql"

	_ "github.com/lib/pq"

	"resourceflow/backend/internal/config"
)

func NewPostgres(cfg config.PostgresConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return db, nil
}
