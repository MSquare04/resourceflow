package handler

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/dto"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) GetHealth(c *echo.Context) error {
	if h.db == nil {
		return c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{
			Success: false,
			Error: dto.APIError{
				Code:    dto.ErrorCodeInternal,
				Message: "PostgreSQL connection is not initialized",
			},
		})
	}

	if err := h.db.PingContext(c.Request().Context()); err != nil {
		slog.Default().Warn("health check failed",
			"operation", "health.ping",
			"request_id", requestID(c),
			"error", err,
		)
		return c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{
			Success: false,
			Error: dto.APIError{
				Code:    dto.ErrorCodeInternal,
				Message: "PostgreSQL is unavailable",
			},
		})
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: map[string]any{
			"status":   "ok",
			"service":  "resourceflow-backend",
			"database": "up",
		},
	})
}
