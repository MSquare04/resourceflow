package handler

import (
	"errors"
	"net/http"
	"time"

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

	from, err := parseOptionalRFC3339Query(c, "from")
	if err != nil {
		return validationError(c, "invalid busy interval range")
	}

	to, err := parseOptionalRFC3339Query(c, "to")
	if err != nil {
		return validationError(c, "invalid busy interval range")
	}

	intervals, err := h.bookings.ListBusyIntervalsByResourceIDInRange(c.Request().Context(), id, from, to)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid resource id")
		case errors.Is(err, service.ErrBusyIntervalRangeInvalid):
			return validationError(c, "invalid busy interval range")
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

func parseOptionalRFC3339Query(c *echo.Context, name string) (*time.Time, error) {
	value := c.QueryParam(name)
	if value == "" {
		return nil, nil
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			utc := parsed.UTC()
			return &utc, nil
		}
	}

	return nil, errors.New("invalid RFC3339 query value")
}
