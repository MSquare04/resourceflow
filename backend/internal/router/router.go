package router

import (
	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/handler"
	rfmiddleware "resourceflow/backend/internal/middleware"
)

type Dependencies struct {
	HealthHandler     *handler.HealthHandler
	AuthHandler       *handler.AuthHandler
	DepartmentHandler *handler.DepartmentHandler
	UserHandler       *handler.UserHandler
	AuthMiddleware    *rfmiddleware.AuthMiddleware
}

func Register(e *echo.Echo, deps Dependencies) {
	e.GET("/health", deps.HealthHandler.GetHealth)

	api := e.Group("/api/v1")
	authGroup := api.Group("/auth")
	authGroup.POST("/login", deps.AuthHandler.Login)
	authGroup.GET("/me", deps.AuthHandler.Me, deps.AuthMiddleware.RequireAuth)

	adminGroup := api.Group("", deps.AuthMiddleware.RequireAuth, rfmiddleware.RequireRoles("admin"))

	adminGroup.POST("/departments", deps.DepartmentHandler.Create)
	adminGroup.GET("/departments", deps.DepartmentHandler.List)
	adminGroup.GET("/departments/:id", deps.DepartmentHandler.GetByID)
	adminGroup.PUT("/departments/:id", deps.DepartmentHandler.Update)

	adminGroup.POST("/users", deps.UserHandler.Create)
	adminGroup.GET("/users", deps.UserHandler.List)
	adminGroup.GET("/users/:id", deps.UserHandler.GetByID)
	adminGroup.PUT("/users/:id", deps.UserHandler.Update)
	adminGroup.PUT("/users/:id/roles", deps.UserHandler.ReplaceRoles)
}
