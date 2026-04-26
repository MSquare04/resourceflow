package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error: dto.APIError{
				Code:    dto.ErrorCodeValidation,
				Message: "invalid request body",
			},
		})
	}

	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)
	if email == "" || password == "" {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error: dto.APIError{
				Code:    dto.ErrorCodeValidation,
				Message: "email and password are required",
			},
		})
	}

	result, err := h.authService.Login(c.Request().Context(), email, password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeUnauthorized,
					Message: "email or password is incorrect",
				},
			})
		case errors.Is(err, service.ErrInactiveUser):
			return c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeForbidden,
					Message: "user account is inactive",
				},
			})
		default:
			return internalError(c, "failed to login", "auth.login", err, "email", email)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: dto.LoginResponse{
			AccessToken: result.AccessToken,
			TokenType:   "Bearer",
			ExpiresAt:   result.ExpiresAt,
			User:        result.User,
		},
	})
}

func (h *AuthHandler) Me(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error: dto.APIError{
				Code:    dto.ErrorCodeUnauthorized,
				Message: "authentication required",
			},
		})
	}

	user, err := h.authService.GetCurrentUser(c.Request().Context(), currentUser.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeUnauthorized,
					Message: "user not found",
				},
			})
		case errors.Is(err, service.ErrInactiveUser):
			return c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    dto.ErrorCodeForbidden,
					Message: "user account is inactive",
				},
			})
		default:
			return internalError(c, "failed to load current user", "auth.me", err, "user_id", currentUser.UserID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: dto.CurrentUserResponse{
			User: user,
		},
	})
}
