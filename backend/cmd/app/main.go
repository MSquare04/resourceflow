package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/config"
	"resourceflow/backend/internal/db"
	"resourceflow/backend/internal/handler"
	rflogger "resourceflow/backend/internal/logger"
	rfmiddleware "resourceflow/backend/internal/middleware"
	"resourceflow/backend/internal/repository"
	"resourceflow/backend/internal/router"
	"resourceflow/backend/internal/service"
)

const expectedMigrationVersion int64 = 5

func main() {
	// Local development convenience: load env from .env if present.
	_ = godotenv.Load(".env", "../.env")

	cfg := config.Load()
	appLogger := rflogger.New(cfg)
	slog.SetDefault(appLogger)

	e := echo.New()
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(rfmiddleware.RequestLogger(appLogger))

	postgres, err := db.NewPostgres(cfg.Postgres)
	if err != nil {
		appLogger.Error("postgres init failed", "error", err)
		os.Exit(1)
	}
	defer postgres.Close()
	if err := postgres.Ping(); err != nil {
		appLogger.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}
	appLogger.Info("postgres connection initialized")
	checkMigrationVersion(postgres, appLogger, expectedMigrationVersion)

	userRepository := repository.NewUserRepository(postgres)
	departmentRepository := repository.NewDepartmentRepository(postgres)
	resourceCategoryRepository := repository.NewResourceCategoryRepository(postgres)
	resourceTypeRepository := repository.NewResourceTypeRepository(postgres)
	resourceRepository := repository.NewResourceRepository(postgres)
	resourceAvailabilityRepository := repository.NewResourceAvailabilityRepository(postgres)
	bookingRuleRepository := repository.NewBookingRuleRepository(postgres)
	passwordHasher := auth.NewBcryptHasher()
	tokenManager := auth.NewTokenManager(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.ExpiresHours)*time.Hour,
	)
	authService := service.NewAuthService(userRepository, passwordHasher, tokenManager)
	departmentService := service.NewDepartmentService(departmentRepository)
	userService := service.NewUserService(userRepository, passwordHasher)
	resourceCategoryService := service.NewResourceCategoryService(resourceCategoryRepository)
	resourceTypeService := service.NewResourceTypeService(resourceTypeRepository)
	resourceService := service.NewResourceService(resourceRepository, resourceTypeRepository)
	resourceAvailabilityService := service.NewResourceAvailabilityService(resourceAvailabilityRepository, resourceRepository)
	bookingRuleService := service.NewBookingRuleService(bookingRuleRepository, resourceTypeRepository)

	router.Register(e, router.Dependencies{
		HealthHandler:               handler.NewHealthHandler(postgres),
		AuthHandler:                 handler.NewAuthHandler(authService),
		DepartmentHandler:           handler.NewDepartmentHandler(departmentService),
		UserHandler:                 handler.NewUserHandler(userService),
		ResourceCategoryHandler:     handler.NewResourceCategoryHandler(resourceCategoryService),
		ResourceTypeHandler:         handler.NewResourceTypeHandler(resourceTypeService),
		ResourceHandler:             handler.NewResourceHandler(resourceService),
		ResourceAvailabilityHandler: handler.NewResourceAvailabilityHandler(resourceAvailabilityService),
		BookingRuleHandler:          handler.NewBookingRuleHandler(bookingRuleService),
		AuthMiddleware:              rfmiddleware.NewAuthMiddleware(tokenManager),
	})

	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
	appLogger.Info("starting http server", "address", addr)
	if err := e.Start(addr); err != nil {
		appLogger.Error("echo server stopped", "error", err)
		os.Exit(1)
	}
}

func checkMigrationVersion(db *sql.DB, logger *slog.Logger, expected int64) {
	var (
		version int64
		dirty   bool
	)

	err := db.QueryRow(`SELECT version, dirty FROM schema_migrations LIMIT 1;`).Scan(&version, &dirty)
	if err != nil {
		logger.Warn("failed to read migration version",
			"expected_version", expected,
			"error", err,
		)
		return
	}

	if dirty {
		logger.Error("database migration state is dirty", "db_version", version, "expected_version", expected)
		return
	}

	if version != expected {
		logger.Warn("database migration version mismatch", "db_version", version, "expected_version", expected)
		return
	}

	logger.Info("database migration version is up to date", "db_version", version)
}
