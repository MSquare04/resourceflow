package middleware

import "github.com/labstack/echo/v5"

func Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// TODO: add JWT/session validation in auth module.
		return next(c)
	}
}
