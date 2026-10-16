// Package response centralises the HTTP error contract of the REST layer.
//
// Errors are never echoed back to the client: the original Go error is logged
// on the server, while the client receives either the curated message of an
// apperrors.UserFacing or the public fallback message of the operation.
package response

import (
	"log"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
)

// Error writes a uniform error response. The original error is logged on the
// server and is NEVER sent to the client. When the error is an
// apperrors.UserFacing its curated message is shown; otherwise the public
// fallback message is used.
func Error(c *echo.Context, status int, fallback string, err error) error {
	if uf, ok := apperrors.Is(err); ok {
		return c.JSON(status, map[string]any{"error": uf.Message})
	}
	if err != nil {
		log.Printf("[api] %s %s -> %d: %v", c.Request().Method, c.Request().URL.Path, status, err)
	}
	return c.JSON(status, map[string]any{"error": fallback})
}

// Fail writes a uniform error response for early returns that carry no
// underlying Go error.
func Fail(c *echo.Context, status int, message string) error {
	return c.JSON(status, map[string]any{"error": message})
}
