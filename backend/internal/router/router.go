package router

import (
	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/handler"
	rfmiddleware "resourceflow/backend/internal/middleware"
)

type Dependencies struct {
	HealthHandler               *handler.HealthHandler
	AuthHandler                 *handler.AuthHandler
	DepartmentHandler           *handler.DepartmentHandler
	UserHandler                 *handler.UserHandler
	ResourceCategoryHandler     *handler.ResourceCategoryHandler
	ResourceTypeHandler         *handler.ResourceTypeHandler
	ResourceHandler             *handler.ResourceHandler
	ResourceAvailabilityHandler *handler.ResourceAvailabilityHandler
	BookingRuleHandler          *handler.BookingRuleHandler
	BookingHandler              *handler.BookingHandler
	AuthMiddleware              *rfmiddleware.AuthMiddleware
}

func Register(e *echo.Echo, deps Dependencies) {
	e.GET("/health", deps.HealthHandler.GetHealth)

	api := e.Group("/api/v1")
	authGroup := api.Group("/auth")
	authGroup.POST("/login", deps.AuthHandler.Login)
	authGroup.GET("/me", deps.AuthHandler.Me, deps.AuthMiddleware.RequireAuth)

	authorizedGroup := api.Group("", deps.AuthMiddleware.RequireAuth)
	adminGroup := authorizedGroup.Group("", rfmiddleware.RequireRoles("admin"))

	adminGroup.POST("/departments", deps.DepartmentHandler.Create)
	adminGroup.GET("/departments", deps.DepartmentHandler.List)
	adminGroup.GET("/departments/:id", deps.DepartmentHandler.GetByID)
	adminGroup.PUT("/departments/:id", deps.DepartmentHandler.Update)

	adminGroup.POST("/users", deps.UserHandler.Create)
	adminGroup.GET("/users", deps.UserHandler.List)
	adminGroup.GET("/users/:id", deps.UserHandler.GetByID)
	adminGroup.PUT("/users/:id", deps.UserHandler.Update)
	adminGroup.PUT("/users/:id/roles", deps.UserHandler.ReplaceRoles)

	adminGroup.POST("/resource-categories", deps.ResourceCategoryHandler.Create)
	adminGroup.PUT("/resource-categories/:id", deps.ResourceCategoryHandler.Update)
	authorizedGroup.GET("/resource-categories", deps.ResourceCategoryHandler.List)
	authorizedGroup.GET("/resource-categories/:id", deps.ResourceCategoryHandler.GetByID)

	adminGroup.POST("/resource-types", deps.ResourceTypeHandler.Create)
	adminGroup.PUT("/resource-types/:id", deps.ResourceTypeHandler.Update)
	authorizedGroup.GET("/resource-types", deps.ResourceTypeHandler.List)
	authorizedGroup.GET("/resource-types/:id", deps.ResourceTypeHandler.GetByID)

	adminGroup.POST("/resources", deps.ResourceHandler.Create)
	adminGroup.PUT("/resources/:id", deps.ResourceHandler.Update)
	authorizedGroup.GET("/resources", deps.ResourceHandler.List)
	authorizedGroup.GET("/resources/:id", deps.ResourceHandler.GetByID)

	adminGroup.POST("/resources/:id/availability", deps.ResourceAvailabilityHandler.Create)
	adminGroup.PUT("/resources/:id/availability/:availabilityId", deps.ResourceAvailabilityHandler.Update)
	adminGroup.DELETE("/resources/:id/availability/:availabilityId", deps.ResourceAvailabilityHandler.Delete)
	authorizedGroup.GET("/resources/:id/availability", deps.ResourceAvailabilityHandler.ListByResourceID)
	authorizedGroup.GET("/resources/:id/availability/:availabilityId", deps.ResourceAvailabilityHandler.GetByID)

	adminGroup.POST("/booking-rules", deps.BookingRuleHandler.Create)
	adminGroup.PUT("/booking-rules/:id", deps.BookingRuleHandler.Update)
	authorizedGroup.GET("/booking-rules", deps.BookingRuleHandler.List)
	authorizedGroup.GET("/booking-rules/:id", deps.BookingRuleHandler.GetByID)

	authorizedGroup.POST("/bookings", deps.BookingHandler.Create)
	authorizedGroup.GET("/bookings", deps.BookingHandler.List)
	authorizedGroup.GET("/bookings/:id", deps.BookingHandler.GetByID)
	authorizedGroup.GET("/my/bookings", deps.BookingHandler.MyList)
	authorizedGroup.POST("/bookings/:id/cancel", deps.BookingHandler.Cancel)
	authorizedGroup.POST("/bookings/:id/approve", deps.BookingHandler.Approve)
	authorizedGroup.POST("/bookings/:id/reject", deps.BookingHandler.Reject)
	authorizedGroup.POST("/bookings/:id/complete", deps.BookingHandler.Complete)
}
