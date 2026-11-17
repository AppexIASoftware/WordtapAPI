package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
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

type stubGlobalStudentsReader struct {
	lastFilter dashQuery.GlobalStudentFilter
}

func (s *stubGlobalStudentsReader) ListGlobalStudents(_ context.Context, filter dashQuery.GlobalStudentFilter) (*dashQuery.GlobalStudentsResponse, error) {
	s.lastFilter = filter
	return &dashQuery.GlobalStudentsResponse{
		Items:   []dashQuery.GlobalStudentItem{},
		Total:   0,
		Courses: []dashQuery.GlobalCourseOption{},
	}, nil
}

type stubDashboardSettingRepo struct {
	rbacJSON string
}

func (s *stubDashboardSettingRepo) Get(_ context.Context, key string) (*entities.PlatformSetting, error) {
	if key == access.RBACSettingKey && s.rbacJSON != "" {
		return &entities.PlatformSetting{Key: key, Value: s.rbacJSON}, nil
	}
	return nil, nil
}
func (s *stubDashboardSettingRepo) Set(context.Context, string, string) error { return nil }
func (s *stubDashboardSettingRepo) GetAll(context.Context) ([]entities.PlatformSetting, error) {
	return nil, nil
}

func TestDashboardRouter_ListGlobalStudents(t *testing.T) {
	settingRepo := &stubDashboardSettingRepo{}
	eval := access.NewRBACPolicyEvaluator(settingRepo)
	studentsReader := &stubGlobalStudentsReader{}

	e := echo.New()
	v1 := e.Group("/api/v1")

	injectHeaders := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", c.Request().Header.Get("X-Test-User-ID"))
			c.Set("user_role", c.Request().Header.Get("X-Test-Role"))
			return next(c)
		}
	}
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)
	globalStudentsAccess := restMiddleware.RequirePermissionOrRole(eval, access.PermStudentsViewDirectory, entities.RoleInstructor)

	router := NewDashboardRouter(dashQuery.NewGetAdminDashboardHandler(&stubDashboardReader{})).
		WithGlobalStudentsHandler(dashQuery.NewListGlobalStudentsHandler(studentsReader)).
		WithRBACEvaluator(eval)
	router.RegisterTeacherRoutes(v1, injectHeaders, instructorOrAdmin, globalStudentsAccess)

	doReq := func(userID, role string) *httptest.ResponseRecorder {
		studentsReader.lastFilter = dashQuery.GlobalStudentFilter{InstructorID: "sentinel"}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/global-students", nil)
		req.Header.Set("X-Test-User-ID", userID)
		req.Header.Set("X-Test-Role", role)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	t.Run("admin receives 200 with unscoped filter", func(t *testing.T) {
		rec := doReq("admin-1", "admin")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if studentsReader.lastFilter.InstructorID != "" {
			t.Fatalf("expected empty InstructorID for admin, got %q", studentsReader.lastFilter.InstructorID)
		}
	})

	t.Run("moderator without students:view_directory receives 403", func(t *testing.T) {
		settingRepo.rbacJSON = ""
		rec := doReq("mod-1", "moderator")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("moderator with students:view_directory receives 200 unscoped", func(t *testing.T) {
		settingRepo.rbacJSON = `{"students:view_directory":{"moderator":true}}`
		rec := doReq("mod-1", "moderator")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if studentsReader.lastFilter.InstructorID != "" {
			t.Fatalf("expected empty InstructorID for authorized moderator, got %q", studentsReader.lastFilter.InstructorID)
		}
	})

	t.Run("instructor receives 200 strictly scoped to own instructor id", func(t *testing.T) {
		settingRepo.rbacJSON = ""
		rec := doReq("teacher-77", "instructor")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if studentsReader.lastFilter.InstructorID != "teacher-77" {
			t.Fatalf("expected InstructorID=teacher-77, got %q", studentsReader.lastFilter.InstructorID)
		}
	})

	t.Run("student receives 403", func(t *testing.T) {
		rec := doReq("student-1", "student")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
