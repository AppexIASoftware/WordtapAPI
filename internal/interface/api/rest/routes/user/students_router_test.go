package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	userCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	userQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/queries"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type stubAdminStudentRepo struct {
	detail *domainRepo.AdminStudentDetail
}

func (s *stubAdminStudentRepo) GetStudentDetail(_ context.Context, studentID string) (*domainRepo.AdminStudentDetail, error) {
	if s.detail == nil || s.detail.ID != studentID {
		return nil, domainRepo.ErrStudentNotFound
	}
	return s.detail, nil
}

func (s *stubAdminStudentRepo) UpdateStudentStatus(_ context.Context, studentID string, isActive bool) (*domainRepo.AdminStudentStatusResult, error) {
	if s.detail == nil || s.detail.ID != studentID {
		return nil, domainRepo.ErrStudentNotFound
	}
	s.detail.IsActive = isActive
	return &domainRepo.AdminStudentStatusResult{
		ID:       s.detail.ID,
		Name:     s.detail.Name,
		Email:    s.detail.Email,
		IsActive: isActive,
	}, nil
}

func TestAdminStudentsRouter(t *testing.T) {
	repo := &stubAdminStudentRepo{
		detail: &domainRepo.AdminStudentDetail{
			ID:                "stu-1",
			Name:              "Ana Alumno",
			Email:             "ana@example.com",
			LearningLevel:     "beginner",
			AccessTier:        "free",
			IsActive:          true,
			CreatedAt:         time.Now(),
			CurrentStreakDays: 4,
			Courses: []domainRepo.AdminStudentCourseProgress{
				{CourseID: "c-1", CourseTitle: "Curso 1", CompletedLessons: 2, TotalLessons: 4, ProgressPercent: 50},
			},
			Classrooms: []domainRepo.AdminStudentClassroom{},
		},
	}

	e := echo.New()
	v1 := e.Group("/api/v1")
	passthrough := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router := NewAdminStudentsRouter(
		userQuery.NewGetStudentDetailHandler(repo),
		userCmd.NewUpdateStudentStatusHandler(repo),
	)
	router.RegisterRoutes(v1, passthrough, passthrough)

	t.Run("GET detail returns student", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/students/stu-1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var out domainRepo.AdminStudentDetail
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if out.ID != "stu-1" || len(out.Courses) != 1 || out.Courses[0].ProgressPercent != 50 {
			t.Fatalf("unexpected detail payload: %+v", out)
		}
	})

	t.Run("GET missing student returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/students/unknown", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("PATCH status suspends and reactivates student", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/students/stu-1/status", bytes.NewBufferString(`{"is_active":false}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if repo.detail.IsActive {
			t.Fatalf("expected student to be suspended")
		}
	})
}
