package router

import (
	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/handler"
	rfmiddleware "resourceflow/backend/internal/middleware"
)

type Dependencies struct {
	HealthHandler *handler.HealthHandler
}

func Register(e *echo.Echo, deps Dependencies) {
	e.GET("/health", deps.HealthHandler.GetHealth)

	api := e.Group("/api/v1")
	_ = api.Group("", rfmiddleware.Auth)
}
