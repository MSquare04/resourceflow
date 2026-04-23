package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"resourceflow/backend/internal/config"
	"resourceflow/backend/internal/db"
	"resourceflow/backend/internal/handler"
	"resourceflow/backend/internal/router"
)

func main() {
	// Local development convenience: load env from .env if present.
	_ = godotenv.Load(".env", "../.env")

	cfg := config.Load()

	e := echo.New()
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())

	postgres, err := db.NewPostgres(cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres init failed: %v", err)
	}
	defer postgres.Close()

	router.Register(e, router.Dependencies{
		HealthHandler: handler.NewHealthHandler(postgres),
	})

	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
	if err := e.Start(addr); err != nil {
		log.Fatalf("echo server stopped: %v", err)
	}
}
