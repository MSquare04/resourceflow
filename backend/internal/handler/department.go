package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type DepartmentHandler struct {
	departments *service.DepartmentService
}

func NewDepartmentHandler(departments *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{departments: departments}
}

func (h *DepartmentHandler) Create(c *echo.Context) error {
	var req dto.CreateDepartmentRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	department, err := h.departments.Create(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "name is required")
		case errors.Is(err, service.ErrDepartmentNameExists):
			return conflictError(c, "department name already exists")
		default:
			return internalError(c, "failed to create department")
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    department,
	})
}

func (h *DepartmentHandler) List(c *echo.Context) error {
	departments, err := h.departments.List(c.Request().Context())
	if err != nil {
		return internalError(c, "failed to load departments")
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    departments,
	})
}

func (h *DepartmentHandler) GetByID(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid department id")
	}

	department, err := h.departments.GetByID(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDepartmentNotFound):
			return notFoundError(c, "department not found")
		default:
			return internalError(c, "failed to load department")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    department,
	})
}

func (h *DepartmentHandler) Update(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid department id")
	}

	var req dto.UpdateDepartmentRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	department, err := h.departments.Update(c.Request().Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "name is required")
		case errors.Is(err, service.ErrDepartmentNotFound):
			return notFoundError(c, "department not found")
		case errors.Is(err, service.ErrDepartmentNameExists):
			return conflictError(c, "department name already exists")
		default:
			return internalError(c, "failed to update department")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    department,
	})
}

func parseIDParam(c *echo.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

func validationError(c *echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    "validation_error",
			Message: message,
		},
	})
}

func notFoundError(c *echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    "not_found",
			Message: message,
		},
	})
}

func conflictError(c *echo.Context, message string) error {
	return c.JSON(http.StatusConflict, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    "conflict",
			Message: message,
		},
	})
}

func internalError(c *echo.Context, message string) error {
	return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    "internal_error",
			Message: message,
		},
	})
}
