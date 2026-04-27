package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

type BookingHandler struct {
	bookings *service.BookingService
}

func NewBookingHandler(bookings *service.BookingService) *BookingHandler {
	return &BookingHandler{bookings: bookings}
}

func (h *BookingHandler) Create(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	var req dto.CreateBookingRequest
	if err := c.Bind(&req); err != nil {
		return validationError(c, "invalid request body")
	}

	booking, err := h.bookings.Create(ctx, currentUser.UserID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid booking payload")
		case errors.Is(err, service.ErrResourceNotFound):
			return notFoundError(c, "resource not found")
		case errors.Is(err, service.ErrBookingOutOfAvailability):
			return validationError(c, "booking interval is outside resource availability")
		case errors.Is(err, service.ErrBookingRuleNotConfigured):
			return validationError(c, "active booking rule is not configured")
		case errors.Is(err, service.ErrBookingLimitExceeded):
			return validationError(c, "max active bookings per user exceeded")
		case errors.Is(err, service.ErrBookingHorizonExceeded):
			return validationError(c, "booking horizon exceeded")
		case errors.Is(err, service.ErrBookingConflict):
			return conflictError(c, "booking conflicts with existing active booking")
		default:
			return internalError(c, "failed to create booking", "booking.create", err, "user_id", currentUser.UserID, "resource_id", req.ResourceID)
		}
	}

	return c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    booking,
	})
}

func (h *BookingHandler) List(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	isPrivileged := auth.HasAnyRole(currentUser, "admin", "manager")

	var (
		bookings []dto.BookingResponse
		err      error
	)
	if isPrivileged {
		bookings, err = h.bookings.List(ctx)
	} else {
		bookings, err = h.bookings.ListByUserID(ctx, currentUser.UserID)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid booking query")
		default:
			return internalError(c, "failed to load bookings", "booking.list", err, "user_id", currentUser.UserID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    bookings,
	})
}

func (h *BookingHandler) MyList(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	bookings, err := h.bookings.ListByUserID(ctx, currentUser.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid current user")
		default:
			return internalError(c, "failed to load current user bookings", "booking.my_list", err, "user_id", currentUser.UserID)
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    bookings,
	})
}

func (h *BookingHandler) GetByID(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid booking id")
	}

	isPrivileged := auth.HasAnyRole(currentUser, "admin", "manager")

	booking, err := h.bookings.GetByID(ctx, id, isPrivileged)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			return validationError(c, "invalid booking id")
		case errors.Is(err, service.ErrBookingNotFound):
			return notFoundError(c, "booking not found")
		default:
			return internalError(c, "failed to load booking", "booking.get_by_id", err, "booking_id", id, "user_id", currentUser.UserID)
		}
	}

	if !isPrivileged && booking.UserID != currentUser.UserID {
		return notFoundError(c, "booking not found")
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    booking,
	})
}

func (h *BookingHandler) Cancel(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid booking id")
	}

	isPrivileged := auth.HasAnyRole(currentUser, "admin", "manager")
	booking, err := h.bookings.Cancel(ctx, id, currentUser.UserID, isPrivileged)
	if err != nil {
		return handleBookingActionError(c, err, "booking.cancel", id, currentUser.UserID)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    booking,
	})
}

func (h *BookingHandler) Complete(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid booking id")
	}

	isPrivileged := auth.HasAnyRole(currentUser, "admin", "manager")
	booking, err := h.bookings.Complete(ctx, id, currentUser.UserID, isPrivileged)
	if err != nil {
		return handleBookingActionError(c, err, "booking.complete", id, currentUser.UserID)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    booking,
	})
}

func (h *BookingHandler) Approve(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	if !auth.HasAnyRole(currentUser, "admin", "manager") {
		return forbiddenError(c, "insufficient role")
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid booking id")
	}

	booking, err := h.bookings.Approve(ctx, id, currentUser.UserID)
	if err != nil {
		return handleBookingActionError(c, err, "booking.approve", id, currentUser.UserID)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    booking,
	})
}

func (h *BookingHandler) Reject(c *echo.Context) error {
	currentUser, ok := auth.GetCurrentUser(c)
	if !ok {
		return unauthorizedError(c, "authentication required")
	}
	ctx := service.WithRequestID(c.Request().Context(), requestID(c))

	if !auth.HasAnyRole(currentUser, "admin", "manager") {
		return forbiddenError(c, "insufficient role")
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		return validationError(c, "invalid booking id")
	}

	booking, err := h.bookings.Reject(ctx, id, currentUser.UserID)
	if err != nil {
		return handleBookingActionError(c, err, "booking.reject", id, currentUser.UserID)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    booking,
	})
}

func handleBookingActionError(c *echo.Context, err error, operation string, bookingID int64, userID int64) error {
	switch {
	case errors.Is(err, service.ErrValidation):
		return validationError(c, "invalid booking id")
	case errors.Is(err, service.ErrBookingNotFound):
		return notFoundError(c, "booking not found")
	case errors.Is(err, service.ErrBookingInvalidStatusAction):
		return validationError(c, "booking status transition is not allowed")
	default:
		return internalError(c, "failed to process booking action", operation, err, "booking_id", bookingID, "user_id", userID)
	}
}

func unauthorizedError(c *echo.Context, message string) error {
	return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    dto.ErrorCodeUnauthorized,
			Message: message,
		},
	})
}

func forbiddenError(c *echo.Context, message string) error {
	return c.JSON(http.StatusForbidden, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    dto.ErrorCodeForbidden,
			Message: message,
		},
	})
}
