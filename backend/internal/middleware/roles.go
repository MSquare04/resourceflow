package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
)

func RequireRoles(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			currentUser, ok := auth.GetCurrentUser(c)
			if !ok {
				return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
					Success: false,
					Error: dto.APIError{
						Code:    "unauthorized",
						Message: "authentication required",
					},
				})
			}

			if !auth.HasAnyRole(currentUser, roles...) {
				return c.JSON(http.StatusForbidden, dto.ErrorResponse{
					Success: false,
					Error: dto.APIError{
						Code:    "forbidden",
						Message: "insufficient role",
					},
				})
			}

			return next(c)
		}
	}
}
