package course_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/course"
)

type mockCourseRepository struct {
	courses []entities.Course
}

func (m *mockCourseRepository) Create(ctx context.Context, c *entities.Course) error {
	m.courses = append(m.courses, *c)
	return nil
}

func (m *mockCourseRepository) FindByAuthor(ctx context.Context, authorID string) ([]entities.Course, error) {
	var result []entities.Course
	for _, c := range m.courses {
		if c.CreatedBy != nil && *c.CreatedBy == authorID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCourseRepository) FindByID(ctx context.Context, id string) (*entities.Course, error) {
	for _, c := range m.courses {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, errors.New("record not found")
}

func (m *mockCourseRepository) Update(ctx context.Context, c *entities.Course) error {
	for i, existing := range m.courses {
		if existing.ID == c.ID {
			m.courses[i] = *c
			return nil
		}
	}
	return errors.New("record not found")
}

func setupTestServer() (*echo.Echo, *security.JWTService, *mockCourseRepository) {
	e := echo.New()
	jwtSvc := security.NewJWTService("test-secret-key-32-bytes-minimum!", 15*time.Minute, time.Hour)
	repo := &mockCourseRepository{}

	createCmd := courseCmd.NewCreateCourseHandler(repo)
	updateCmd := courseCmd.NewUpdateCourseHandler(repo)
	submitCmd := courseCmd.NewSubmitCourseReviewHandler(repo)
	listQuery := courseQuery.NewListTeacherCoursesHandler(repo)
	getQuery := courseQuery.NewGetCourseDetailHandler(repo)

	router := course.NewCourseRouter(createCmd, updateCmd, submitCmd, listQuery, getQuery)

	authRequired := restMiddleware.RequireAuth(jwtSvc)
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)

	v1 := e.Group("/api/v1")
	router.RegisterRoutes(v1, authRequired, instructorOrAdmin)

	return e, jwtSvc, repo
}

func TestCourseRoutesRBAC(t *testing.T) {
	e, jwtSvc, repo := setupTestServer()

	teacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "teacher-123",
		Email:      "teacher@wordtap.app",
		Role:       entities.RoleInstructor,
		AccessTier: entities.AccessTierCourse,
	})

	otherTeacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "teacher-999",
		Email:      "other@wordtap.app",
		Role:       entities.RoleInstructor,
		AccessTier: entities.AccessTierCourse,
	})

	studentToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "student-456",
		Email:      "student@wordtap.app",
		Role:       entities.RoleStudent,
		AccessTier: entities.AccessTierFree,
	})

	var createdID string

	t.Run("student cannot create course (403 Forbidden)", func(t *testing.T) {
		body := `{"title":"Curso Prohibido"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+studentToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 Forbidden, got %d", rec.Code)
		}
	})

	t.Run("teacher can create course (201 Created)", func(t *testing.T) {
		body := `{"title":"Curso Permitido","level":"A1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var created entities.Course
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		createdID = created.ID
		if created.Title != "Curso Permitido" {
			t.Errorf("expected title 'Curso Permitido', got '%s'", created.Title)
		}
		if created.Status != entities.ContentStatusDraft {
			t.Errorf("expected status 'draft', got '%s'", created.Status)
		}
	})

	t.Run("get course detail (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+createdID, nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", rec.Code)
		}
	})

	t.Run("other teacher cannot edit course (400 Bad Request / Forbidden)", func(t *testing.T) {
		body := `{"title":"Intento de hack"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+createdID, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+otherTeacherToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for forbidden author edit, got %d", rec.Code)
		}
	})

	t.Run("author can update course (200 OK)", func(t *testing.T) {
		body := `{"title":"Curso Actualizado","level":"A2"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+createdID, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		if len(repo.courses) > 0 && repo.courses[0].Title != "Curso Actualizado" {
			t.Errorf("expected updated title, got %s", repo.courses[0].Title)
		}
	})

	t.Run("other teacher cannot submit course for review (400 Bad Request)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+createdID+"/submit-review", nil)
		req.Header.Set("Authorization", "Bearer "+otherTeacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 when submitting other's course, got %d", rec.Code)
		}
	})

	t.Run("author can submit course for review (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+createdID+"/submit-review", nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK when submitting for review, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var submitted entities.Course
		_ = json.Unmarshal(rec.Body.Bytes(), &submitted)
		if submitted.Status != entities.ContentStatusInReview {
			t.Errorf("expected status 'in_review', got '%s'", submitted.Status)
		}
	})

	t.Run("cannot resubmit already submitted course (400 Bad Request)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+createdID+"/submit-review", nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 on duplicate submit, got %d", rec.Code)
		}
	})
}
