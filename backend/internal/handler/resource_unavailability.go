package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type ResourceUnavailabilityHandler struct {
	unavailability *service.ResourceUnavailabilityService
}

func NewResourceUnavailabilityHandler(unavailability *service.ResourceUnavailabilityService) *ResourceUnavailabilityHandler {
	return &ResourceUnavailabilityHandler{unavailability: unavailability}
}

func (h *ResourceUnavailabilityHandler) Create(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}

	var req dto.CreateResourceUnavailabilityRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	item, err := h.unavailability.Create(c.Request().Context(), resourceID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource unavailability payload")
		case errors.Is(err, service.ErrUnavailabilityConflict):
			return conflictError(c, "resource unavailability conflicts with existing active bookings")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		default:
			return internalError(c, "failed to create resource unavailability", "resource_unavailability.create", err, "resource_id", resourceID)
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    item,
	})
}

func (h *ResourceUnavailabilityHandler) ListByResourceID(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}

	items, err := h.unavailability.ListByResourceID(c.Request().Context(), resourceID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource id")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		default:
			return internalError(c, "failed to load resource unavailability", "resource_unavailability.list", err, "resource_id", resourceID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    items,
	})
}

func (h *ResourceUnavailabilityHandler) GetByID(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}
	unavailabilityID, err := parseIDParam(c, "unavailabilityId")
	if err != nil {
		return validationError(c, "invalid unavailability id")
	}

	item, err := h.unavailability.GetByID(c.Request().Context(), resourceID, unavailabilityID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid ids")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		case errors.Is(err, service.ErrResourceUnavailabilityNotFound):
			return notFoundError(c, "resource unavailability not found")
		default:
			return internalError(c, "failed to load resource unavailability", "resource_unavailability.get_by_id", err, "resource_id", resourceID, "unavailability_id", unavailabilityID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    item,
	})
}

func (h *ResourceUnavailabilityHandler) Update(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}
	unavailabilityID, err := parseIDParam(c, "unavailabilityId")
	if err != nil {
		return validationError(c, "invalid unavailability id")
	}

	var req dto.UpdateResourceUnavailabilityRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	item, err := h.unavailability.Update(c.Request().Context(), resourceID, unavailabilityID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource unavailability payload")
		case errors.Is(err, service.ErrUnavailabilityConflict):
			return conflictError(c, "resource unavailability conflicts with existing active bookings")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		case errors.Is(err, service.ErrResourceUnavailabilityNotFound):
			return notFoundError(c, "resource unavailability not found")
		default:
			return internalError(c, "failed to update resource unavailability", "resource_unavailability.update", err, "resource_id", resourceID, "unavailability_id", unavailabilityID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    item,
	})
}

func (h *ResourceUnavailabilityHandler) Delete(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}
	unavailabilityID, err := parseIDParam(c, "unavailabilityId")
	if err != nil {
		return validationError(c, "invalid unavailability id")
	}

	if err := h.unavailability.Delete(c.Request().Context(), resourceID, unavailabilityID); err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid ids")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		case errors.Is(err, service.ErrResourceUnavailabilityNotFound):
			return notFoundError(c, "resource unavailability not found")
		default:
			return internalError(c, "failed to delete resource unavailability", "resource_unavailability.delete", err, "resource_id", resourceID, "unavailability_id", unavailabilityID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: map[string]any{
			"id":      unavailabilityID,
			"deleted": true,
		},
	})
}
