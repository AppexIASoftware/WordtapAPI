package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	dashQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/dashboard/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
)

type stubDashboardReader struct {
	data *dashQuery.AdminDashboardResponse
	err  error
}

func (s *stubDashboardReader) GetAdminDashboard(_ context.Context) (*dashQuery.AdminDashboardResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func TestDashboardRouter_GetAdminDashboard(t *testing.T) {
	e := echo.New()
	v1 := e.Group("/api/v1")

	stub := &stubDashboardReader{
		data: &dashQuery.AdminDashboardResponse{
			Users: dashQuery.DashboardUserStats{TotalUsers: 12, Students: 9, Instructors: 2, Admins: 1},
			Catalog: dashQuery.DashboardCatalogStats{TotalCourses: 5, PublishedCourses: 3, InReviewCourses: 1},
			RecentEvents: []dashQuery.DashboardEventItem{
				{ID: "e1", Action: "course.published", Summary: "Curso B1 publicado", Actor: "Catálogo", CreatedAt: time.Now()},
			},
			GeneratedAt: time.Now(),
		},
	}

	router := NewDashboardRouter(dashQuery.NewGetAdminDashboardHandler(stub))
	allowAll := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	adminOnly := restMiddleware.RequireRole(entities.RoleAdmin)
	router.RegisterRoutes(v1, allowAll, adminOnly)

	t.Run("returns 200 with metrics for admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_role", "admin")

		h := adminOnly(router.GetAdminDashboard)
		if err := h(c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var body dashQuery.AdminDashboardResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if body.Users.TotalUsers != 12 || len(body.RecentEvents) != 1 {
			t.Fatalf("unexpected dashboard payload: %+v", body)
		}
	})

	t.Run("returns 403 for student", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_role", "student")

		h := adminOnly(router.GetAdminDashboard)
		_ = h(c)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})
}
