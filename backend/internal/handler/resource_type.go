package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type ResourceTypeHandler struct {
	resourceTypes *service.ResourceTypeService
}

func NewResourceTypeHandler(resourceTypes *service.ResourceTypeService) *ResourceTypeHandler {
	return &ResourceTypeHandler{resourceTypes: resourceTypes}
}

func (h *ResourceTypeHandler) Create(c *echo.Context) error {
	var req dto.CreateResourceTypeRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	resourceType, err := h.resourceTypes.Create(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "category_id, code and name are required")
		case errors.Is(err, service.ErrResourceTypeCodeExists):
			return conflictError(c, "resource type code already exists")
		default:
			return internalError(c, "failed to create resource type")
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    resourceType,
	})
}

func (h *ResourceTypeHandler) List(c *echo.Context) error {
	resourceTypes, err := h.resourceTypes.List(c.Request().Context())
	if err != nil {
		return internalError(c, "failed to load resource types")
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    resourceTypes,
	})
}

func (h *ResourceTypeHandler) GetByID(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource type id")
	}

	resourceType, err := h.resourceTypes.GetByID(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrResourceTypeNotFound):
			return notFoundError(c, "resource type not found")
		default:
			return internalError(c, "failed to load resource type")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    resourceType,
	})
}

func (h *ResourceTypeHandler) Update(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource type id")
	}

	var req dto.UpdateResourceTypeRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	resourceType, err := h.resourceTypes.Update(c.Request().Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "category_id, code and name are required")
		case errors.Is(err, service.ErrResourceTypeNotFound):
			return notFoundError(c, "resource type not found")
		case errors.Is(err, service.ErrResourceTypeCodeExists):
			return conflictError(c, "resource type code already exists")
		default:
			return internalError(c, "failed to update resource type")
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    resourceType,
	})
}
