package middleware

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/repository"
)

type AuthMiddleware struct {
	tokens *auth.TokenManager
	users  repository.UserRepository
}

func NewAuthMiddleware(tokens *auth.TokenManager, users repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens, users: users}
}

func (m *AuthMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		header := c.Request().Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeUnauthorized,
					Message: "missing or invalid authorization header",
				},
			})
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeUnauthorized,
					Message: "missing access token",
				},
			})
		}

		claims, err := m.tokens.ParseAccessToken(token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeUnauthorized,
					Message: "invalid access token",
				},
			})
		}

		user, err := m.users.FindByID(c.Request().Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
					Success: false,
					Error: dto.APIError{
						Code:    dto.ErrorCodeUnauthorized,
						Message: "user not found",
					},
				})
			}
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeInternal,
					Message: "failed to load current user",
				},
			})
		}
		if !user.IsActive {
			return c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeInactiveUser,
					Message: "user account is inactive",
				},
			})
		}
		if claims.AuthVersion != user.AuthVersion {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeUnauthorized,
					Message: "session is no longer valid",
				},
			})
		}

		roles, err := m.users.ListRolesByUserID(c.Request().Context(), user.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeInternal,
					Message: "failed to load current user roles",
				},
			})
		}

		auth.SetCurrentUser(c, auth.CurrentUser{
			UserID: user.ID,
			Email:  user.Email,
			Roles:  roles,
		})

		return next(c)
	}
}
