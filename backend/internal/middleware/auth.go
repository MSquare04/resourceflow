package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
)

type AuthMiddleware struct {
	tokens *auth.TokenManager
}

func NewAuthMiddleware(tokens *auth.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens}
}

func (m *AuthMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		header := c.Request().Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    "unauthorized",
					Message: "missing or invalid authorization header",
				},
			})
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    "unauthorized",
					Message: "missing access token",
				},
			})
		}

		claims, err := m.tokens.ParseAccessToken(token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    "unauthorized",
					Message: "invalid access token",
				},
			})
		}

		auth.SetCurrentUser(c, auth.CurrentUser{
			UserID: claims.UserID,
			Email:  claims.Email,
			Roles:  claims.Roles,
		})

		return next(c)
	}
}
