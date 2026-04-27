package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type BookingRuleHandler struct {
	bookingRules *service.BookingRuleService
}

func NewBookingRuleHandler(bookingRules *service.BookingRuleService) *BookingRuleHandler {
	return &BookingRuleHandler{bookingRules: bookingRules}
}

func (h *BookingRuleHandler) Create(c *echo.Context) error {
	var req dto.CreateBookingRuleRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	rule, err := h.bookingRules.Create(c.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid booking rule payload")
		default:
			return internalError(c, "failed to create booking rule", "booking_rule.create", err)
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    rule,
	})
}

func (h *BookingRuleHandler) List(c *echo.Context) error {
	rules, err := h.bookingRules.List(c.Request().Context())
	if err != nil {
		return internalError(c, "failed to load booking rules", "booking_rule.list", err)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    rules,
	})
}

func (h *BookingRuleHandler) GetByID(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid booking rule id")
	}

	rule, err := h.bookingRules.GetByID(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid booking rule id")
		case errors.Is(err, service.ErrBookingRuleNotFound):
			return notFoundError(c, "booking rule not found")
		default:
			return internalError(c, "failed to load booking rule", "booking_rule.get_by_id", err, "booking_rule_id", id)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    rule,
	})
}

func (h *BookingRuleHandler) Update(c *echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid booking rule id")
	}

	var req dto.UpdateBookingRuleRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	rule, err := h.bookingRules.Update(c.Request().Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid booking rule payload")
		case errors.Is(err, service.ErrBookingRuleNotFound):
			return notFoundError(c, "booking rule not found")
		default:
			return internalError(c, "failed to update booking rule", "booking_rule.update", err, "booking_rule_id", id)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    rule,
	})
}
