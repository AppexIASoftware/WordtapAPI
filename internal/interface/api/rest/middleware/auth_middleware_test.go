package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
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

func TestRequireAnyPermission(t *testing.T) {
	eval := access.NewRBACPolicyEvaluator(nil)

	t.Run("allows role matching either permission", func(t *testing.T) {
		for _, role := range []entities.UserRole{entities.RoleInstructor, entities.RoleModerator, entities.RoleAdmin} {
			e := echo.New()
			e.GET("/dispatch", func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			}, func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error {
					c.Set("user_role", string(role))
					return next(c)
				}
			}, RequireAnyPermission(eval, access.PermNotificationsClassroomBroadcast, access.PermNotificationsGlobalAnnouncements))

			req := httptest.NewRequest(http.MethodGet, "/dispatch", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("role %s: expected 200 OK, got %d: %s", role, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("rejects role holding neither permission and handles nil evaluator fail-closed", func(t *testing.T) {
		e := echo.New()
		e.GET("/mod-only", func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		}, func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				c.Set("user_role", string(entities.RoleInstructor))
				return next(c)
			}
		}, RequireAnyPermission(eval, access.PermModerationReviewCourses, access.PermModerationManageTeacherApps))

		req := httptest.NewRequest(http.MethodGet, "/mod-only", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for instructor, got %d", rec.Code)
		}

		eNil := echo.New()
		eNil.GET("/nil-eval", func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		}, RequireAnyPermission(nil, access.PermNotificationsClassroomBroadcast))
		recNil := httptest.NewRecorder()
		eNil.ServeHTTP(recNil, httptest.NewRequest(http.MethodGet, "/nil-eval", nil))
		if recNil.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden with nil evaluator, got %d", recNil.Code)
		}
	})
}
