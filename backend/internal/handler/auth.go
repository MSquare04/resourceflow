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
				Code:    "validation_error",
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
				Code:    "validation_error",
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
					Code:    "invalid_credentials",
					Message: "email or password is incorrect",
				},
			})
		case errors.Is(err, service.ErrInactiveUser):
			return c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    "inactive_user",
					Message: "user account is inactive",
				},
			})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    "internal_error",
					Message: "failed to login",
				},
			})
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
				Code:    "unauthorized",
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
					Code:    "unauthorized",
					Message: "user not found",
				},
			})
		case errors.Is(err, service.ErrInactiveUser):
			return c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    "inactive_user",
					Message: "user account is inactive",
				},
			})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Success: false,
				Error: dto.APIError{
					Code:    "internal_error",
					Message: "failed to load current user",
				},
			})
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: dto.CurrentUserResponse{
			User: user,
		},
	})
}
