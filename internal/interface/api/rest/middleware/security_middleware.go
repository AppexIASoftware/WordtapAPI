package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// DefaultBodyLimitBytes caps JSON and API request payloads at 2 MiB.
const DefaultBodyLimitBytes int64 = 2 * 1024 * 1024

// BodyLimit rejects payloads larger than limitBytes with HTTP 413.
func BodyLimit(limitBytes int64) echo.MiddlewareFunc {
	return middleware.BodyLimit(limitBytes)
}

// IPRateLimiter throttles requests per client IP with HTTP 429 when burst capacity is exceeded.
func IPRateLimiter(ratePerSec float64, burst int) echo.MiddlewareFunc {
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      ratePerSec,
		Burst:     burst,
		ExpiresIn: 3 * time.Minute,
	})
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: store,
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			ip := strings.TrimSpace(c.RealIP())
			if ip == "" {
				ip = "unknown"
			}
			return ip, nil
		},
		DenyHandler: func(c *echo.Context, _ string, _ error) error {
			return response.Fail(c, http.StatusTooManyRequests, "Demasiadas solicitudes. Intenta nuevamente en unos segundos.")
		},
		ErrorHandler: func(c *echo.Context, _ error) error {
			return response.Fail(c, http.StatusTooManyRequests, "No se pudo validar el límite de solicitudes.")
		},
	})
}

// NewGormActiveUserChecker verifies users.is_active in MySQL when db is configured.
func NewGormActiveUserChecker(db *gorm.DB) ActiveUserChecker {
	if db == nil {
		return nil
	}
	return func(ctx context.Context, userID string) (bool, error) {
		var row struct {
			IsActive bool
		}
		err := db.WithContext(ctx).
			Table("users").
			Select("is_active").
			Where("id = ?", userID).
			Take(&row).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		return row.IsActive, nil
	}
}
