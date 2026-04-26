package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) Create(c *echo.Context) error {
	var req dto.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	user, err := h.users.Create(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid user payload")
		case errors.Is(err, service.ErrUserEmailExists):
			return conflictError(c, "user email already exists")
		case errors.Is(err, service.ErrRoleCodeNotFound):
			return validationError(c, "one or more role codes are invalid")
		default:
			return internalError(c, "failed to create user")
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    user,
	})
}

func (h *UserHandler) List(c *echo.Context) error {
	users, err := h.users.List(c.Request().Context())
	if err != nil {
		return internalError(c, "failed to load users")
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    users,
	})
}

func (h *UserHandler) GetByID(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid user id")
	}

	user, err := h.users.GetByID(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return notFoundError(c, "user not found")
		default:
			return internalError(c, "failed to load user")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    user,
	})
}

func (h *UserHandler) Update(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid user id")
	}

	var req dto.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	user, err := h.users.Update(c.Request().Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid user payload")
		case errors.Is(err, service.ErrUserNotFound):
			return notFoundError(c, "user not found")
		case errors.Is(err, service.ErrUserEmailExists):
			return conflictError(c, "user email already exists")
		default:
			return internalError(c, "failed to update user")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    user,
	})
}

func (h *UserHandler) ReplaceRoles(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid user id")
	}

	var req dto.UpdateUserRolesRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	user, err := h.users.ReplaceRoles(c.Request().Context(), id, req.Roles)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return notFoundError(c, "user not found")
		case errors.Is(err, service.ErrRoleCodeNotFound):
			return validationError(c, "one or more role codes are invalid")
		default:
			return internalError(c, "failed to update user roles")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    user,
	})
}
