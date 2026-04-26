package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/auth"
)

func RequestLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)
			_, status := echo.ResolveResponseStatus(c.Response(), err)

			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			logArgs := []any{
				"method", c.Request().Method,
				"path", path,
				"status", status,
				"latency_ms", latency.Milliseconds(),
				"request_id", requestID(c),
			}

			if currentUser, ok := auth.GetCurrentUser(c); ok {
				logArgs = append(logArgs, "user_id", currentUser.UserID)
			}

			if err != nil {
				logger.Error("request completed with error", append(logArgs, "error", err)...)
			} else {
				logger.Info("request completed", logArgs...)
			}

			return err
		}
	}
}

func requestID(c *echo.Context) string {
	requestID := c.Response().Header().Get(echo.HeaderXRequestID)
	if requestID != "" {
		return requestID
	}

	return c.Request().Header.Get(echo.HeaderXRequestID)
}
