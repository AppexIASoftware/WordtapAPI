package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
)

func TestRequireAuthBlocksSuspendedUser(t *testing.T) {
	jwtSvc := security.NewJWTService("test-secret-32-bytes-minimum-key!!", 15*time.Minute, 24*time.Hour)
	token, _, err := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "user-1",
		Email:      "stu@example.com",
		Role:       entities.RoleStudent,
		AccessTier: entities.AccessTierFree,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	e := echo.New()
	mw := RequireAuth(jwtSvc, func(_ context.Context, userID string) (bool, error) {
		return userID != "user-1", nil
	})
	e.GET("/protected", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, mw)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for suspended user, got %d: %s", rec.Code, rec.Body.String())
	}
}
