package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type ResourceAvailabilityHandler struct {
	availability *service.ResourceAvailabilityService
}

func NewResourceAvailabilityHandler(availability *service.ResourceAvailabilityService) *ResourceAvailabilityHandler {
	return &ResourceAvailabilityHandler{availability: availability}
}

func (h *ResourceAvailabilityHandler) Create(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}

	var req dto.CreateResourceAvailabilityRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	availability, err := h.availability.Create(c.Request().Context(), resourceID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid availability payload")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		default:
			return internalError(c, "failed to create resource availability", "resource_availability.create", err, "resource_id", resourceID)
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    availability,
	})
}

func (h *ResourceAvailabilityHandler) ListByResourceID(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}

	availabilityList, err := h.availability.ListByResourceID(c.Request().Context(), resourceID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource id")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		default:
			return internalError(c, "failed to load resource availability", "resource_availability.list", err, "resource_id", resourceID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    availabilityList,
	})
}

func (h *ResourceAvailabilityHandler) GetByID(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}
	availabilityID, err := parseIDParam(c, "availabilityId")
	if err != nil {
		return validationError(c, "invalid availability id")
	}

	availability, err := h.availability.GetByID(c.Request().Context(), resourceID, availabilityID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid ids")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		case errors.Is(err, service.ErrResourceAvailabilityNotFound):
			return notFoundError(c, "resource availability not found")
		default:
			return internalError(
				c,
				"failed to load resource availability",
				"resource_availability.get_by_id",
				err,
				"resource_id", resourceID,
				"availability_id", availabilityID,
			)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    availability,
	})
}

func (h *ResourceAvailabilityHandler) Update(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}
	availabilityID, err := parseIDParam(c, "availabilityId")
	if err != nil {
		return validationError(c, "invalid availability id")
	}

	var req dto.UpdateResourceAvailabilityRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	availability, err := h.availability.Update(c.Request().Context(), resourceID, availabilityID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid availability payload")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		case errors.Is(err, service.ErrResourceAvailabilityNotFound):
			return notFoundError(c, "resource availability not found")
		default:
			return internalError(
				c,
				"failed to update resource availability",
				"resource_availability.update",
				err,
				"resource_id", resourceID,
				"availability_id", availabilityID,
			)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    availability,
	})
}

func (h *ResourceAvailabilityHandler) Delete(c *echo.Context) error {
	resourceID, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}
	availabilityID, err := parseIDParam(c, "availabilityId")
	if err != nil {
		return validationError(c, "invalid availability id")
	}

	err = h.availability.Delete(c.Request().Context(), resourceID, availabilityID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid ids")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		case errors.Is(err, service.ErrResourceAvailabilityNotFound):
			return notFoundError(c, "resource availability not found")
		default:
			return internalError(
				c,
				"failed to delete resource availability",
				"resource_availability.delete",
				err,
				"resource_id", resourceID,
				"availability_id", availabilityID,
			)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: map[string]any{
			"id":      availabilityID,
			"deleted": true,
		},
	})
}
