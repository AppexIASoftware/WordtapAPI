package course

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/labstack/echo/v5"
)

type accessRouteStub struct{ user, course string }

func (s *accessRouteStub) GrantTestAccess(_ context.Context, user, course string) error {
	s.user, s.course = user, course
	return nil
}

func (s *accessRouteStub) HasTestAccess(_ context.Context, user, course string, enabled bool) (bool, error) {
	s.user, s.course = user, course
	return enabled && user == "student-1" && course == "course-1", nil
}

func TestCourseAccessReadUsesAuthenticatedIdentity(t *testing.T) {
	jwt := security.NewJWTService("test-course-access-secret", time.Hour, time.Hour)
	student, _, _ := jwt.GenerateAccessToken(&entities.User{ID: "student-1", Role: entities.RoleStudent})
	repo := &accessRouteStub{}
	e := echo.New()
	NewCourseAccessRouter(commands.NewGrantTestCourseAccessHandler(repo, true)).WithAccessReader(repo, true).RegisterRoutes(e.Group("/api/v1"), middleware.RequireAuth(jwt), middleware.RequireRole(entities.RoleStudent))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/student/courses/course-1/test-access?user_id=other", nil)
	req.Header.Set("Authorization", "Bearer "+student)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || repo.user != "student-1" || !strings.Contains(rec.Body.String(), `"has_access":true`) {
		t.Fatalf("unexpected response: %d %s, user %q", rec.Code, rec.Body.String(), repo.user)
	}
}

func TestTestCourseAccessHTTP(t *testing.T) {
	jwt := security.NewJWTService("test-course-access-secret", time.Hour, time.Hour)
	student, _, _ := jwt.GenerateAccessToken(&entities.User{ID: "student-1", Role: entities.RoleStudent})
	teacher, _, _ := jwt.GenerateAccessToken(&entities.User{ID: "teacher-1", Role: entities.RoleInstructor})
	for _, tc := range []struct {
		name, token string
		enabled     bool
		status      int
	}{
		{"anonymous", "", true, http.StatusUnauthorized},
		{"teacher", teacher, true, http.StatusForbidden},
		{"disabled", student, false, http.StatusNotFound},
		{"student", student, true, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &accessRouteStub{}
			e := echo.New()
			NewCourseAccessRouter(commands.NewGrantTestCourseAccessHandler(repo, tc.enabled)).RegisterRoutes(e.Group("/api/v1"), middleware.RequireAuth(jwt), middleware.RequireRole(entities.RoleStudent))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/student/courses/course-1/test-access", strings.NewReader(`{"user_id":"another-student"}`))
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			if tc.status == http.StatusOK && (repo.user != "student-1" || repo.course != "course-1") {
				t.Fatalf("unexpected target: %+v", repo)
			}
			if tc.status != http.StatusOK && repo.user != "" {
				t.Fatal("rejected request reached repository")
			}
		})
	}
}
