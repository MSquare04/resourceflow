package config

import (
	"fmt"

	"resourceflow/backend/internal/utils"
)

type Config struct {
	Env      string
	HTTP     HTTPConfig
	Postgres PostgresConfig
}

type HTTPConfig struct {
	Host string
	Port int
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host,
		p.Port,
		p.User,
		p.Password,
		p.DBName,
		p.SSLMode,
	)
}

func Load() Config {
	return Config{
		Env: utils.GetEnv("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Host: utils.GetEnv("HTTP_HOST", "127.0.0.1"),
			Port: utils.GetEnvInt("HTTP_PORT", 18080),
		},
		Postgres: PostgresConfig{
			Host:     utils.GetEnv("POSTGRES_HOST", "127.0.0.1"),
			Port:     utils.GetEnvInt("POSTGRES_PORT", 5432),
			User:     utils.GetEnv("POSTGRES_USER", "postgres"),
			Password: utils.GetEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:   utils.GetEnv("POSTGRES_DB", "resourceflow"),
			SSLMode:  utils.GetEnv("POSTGRES_SSLMODE", "disable"),
		},
	}
}
