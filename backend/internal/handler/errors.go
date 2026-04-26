package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
)

func parseIDParam(c *echo.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

func validationError(c *echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    dto.ErrorCodeValidation,
			Message: message,
		},
	})
}

func notFoundError(c *echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    dto.ErrorCodeNotFound,
			Message: message,
		},
	})
}

func conflictError(c *echo.Context, message string) error {
	return c.JSON(http.StatusConflict, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    dto.ErrorCodeConflict,
			Message: message,
		},
	})
}

func internalError(c *echo.Context, message, operation string, err error, attrs ...any) error {
	logArgs := []any{
		"operation", operation,
		"method", c.Request().Method,
		"path", c.Path(),
		"request_id", requestID(c),
		"error", err,
	}
	logArgs = append(logArgs, attrs...)

	slog.Default().Error("internal server error", logArgs...)

	return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
		Success: false,
		Error: dto.APIError{
			Code:    dto.ErrorCodeInternal,
			Message: message,
		},
	})
}

func requestID(c *echo.Context) string {
	id := c.Response().Header().Get(echo.HeaderXRequestID)
	if id != "" {
		return id
	}
	return c.Request().Header.Get(echo.HeaderXRequestID)
}
