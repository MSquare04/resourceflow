package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type ResourceCategoryHandler struct {
	categories *service.ResourceCategoryService
}

func NewResourceCategoryHandler(categories *service.ResourceCategoryService) *ResourceCategoryHandler {
	return &ResourceCategoryHandler{categories: categories}
}

func (h *ResourceCategoryHandler) Create(c *echo.Context) error {
	var req dto.CreateResourceCategoryRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	category, err := h.categories.Create(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "code and name are required")
		case errors.Is(err, service.ErrResourceCategoryCodeExists):
			return conflictError(c, "resource category code already exists")
		default:
			return internalError(c, "failed to create resource category")
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    category,
	})
}

func (h *ResourceCategoryHandler) List(c *echo.Context) error {
	categories, err := h.categories.List(c.Request().Context())
	if err != nil {
		return internalError(c, "failed to load resource categories")
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    categories,
	})
}

func (h *ResourceCategoryHandler) GetByID(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource category id")
	}

	category, err := h.categories.GetByID(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrResourceCategoryNotFound):
			return notFoundError(c, "resource category not found")
		default:
			return internalError(c, "failed to load resource category")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    category,
	})
}

func (h *ResourceCategoryHandler) Update(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource category id")
	}

	var req dto.UpdateResourceCategoryRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	category, err := h.categories.Update(c.Request().Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "code and name are required")
		case errors.Is(err, service.ErrResourceCategoryNotFound):
			return notFoundError(c, "resource category not found")
		case errors.Is(err, service.ErrResourceCategoryCodeExists):
			return conflictError(c, "resource category code already exists")
		default:
			return internalError(c, "failed to update resource category")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    category,
	})
}
