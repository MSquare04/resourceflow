package config

import (
	"fmt"
	"time"

	"resourceflow/backend/internal/utils"
)

type Config struct {
	Env      string
	LogLevel string
	Timezone string
	Location *time.Location
	HTTP     HTTPConfig
	Postgres PostgresConfig
	JWT      JWTConfig
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

type JWTConfig struct {
	Secret       string
	ExpiresHours int
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
	jwtExpiresHours := utils.GetEnvInt("JWT_EXPIRES_HOURS", 24)
	if jwtExpiresHours <= 0 {
		jwtExpiresHours = 24
	}

	timezoneName := utils.GetEnv("APP_TIMEZONE", "UTC")
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		timezoneName = "UTC"
		location = time.UTC
	}

	return Config{
		Env:      utils.GetEnv("APP_ENV", "development"),
		LogLevel: utils.GetEnv("LOG_LEVEL", "info"),
		Timezone: timezoneName,
		Location: location,
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
		JWT: JWTConfig{
			Secret:       utils.GetEnv("JWT_SECRET", "change-me"),
			ExpiresHours: jwtExpiresHours,
		},
	}
}
