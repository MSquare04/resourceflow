package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type ResourceHandler struct {
	resources *service.ResourceService
	bookings  *service.BookingService
}

func NewResourceHandler(resources *service.ResourceService, bookings *service.BookingService) *ResourceHandler {
	return &ResourceHandler{
		resources: resources,
		bookings:  bookings,
	}
}

func (h *ResourceHandler) Create(c *echo.Context) error {
	var req dto.CreateResourceRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	resource, err := h.resources.Create(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource payload")
		default:
			return internalError(c, "failed to create resource", "resource.create", err)
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    resource,
	})
}

func (h *ResourceHandler) List(c *echo.Context) error {
	resources, err := h.resources.List(c.Request().Context())
	if err != nil {
		return internalError(c, "failed to load resources", "resource.list", err)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    resources,
	})
}

func (h *ResourceHandler) GetByID(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}

	resource, err := h.resources.GetByID(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		default:
			return internalError(c, "failed to load resource", "resource.get_by_id", err, "resource_id", id)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    resource,
	})
}

func (h *ResourceHandler) Update(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}

	var req dto.UpdateResourceRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	resource, err := h.resources.Update(c.Request().Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource payload")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		default:
			return internalError(c, "failed to update resource", "resource.update", err, "resource_id", id)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    resource,
	})
}

func (h *ResourceHandler) ListBusyIntervals(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid resource id")
	}

	intervals, err := h.bookings.ListBusyIntervalsByResourceID(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource id")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		default:
			return internalError(c, "failed to load busy intervals", "resource.list_busy_intervals", err, "resource_id", id)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    intervals,
	})
}
