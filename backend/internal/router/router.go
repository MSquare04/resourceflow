package router

import (
	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/handler"
	rfmiddleware "resourceflow/backend/internal/middleware"
)

type Dependencies struct {
	HealthHandler  *handler.HealthHandler
	AuthHandler    *handler.AuthHandler
	AuthMiddleware *rfmiddleware.AuthMiddleware
}

func Register(e *echo.Echo, deps Dependencies) {
	e.GET("/health", deps.HealthHandler.GetHealth)

	api := e.Group("/api/v1")
	authGroup := api.Group("/auth")
	authGroup.POST("/login", deps.AuthHandler.Login)
	authGroup.GET("/me", deps.AuthHandler.Me, deps.AuthMiddleware.RequireAuth)
}
