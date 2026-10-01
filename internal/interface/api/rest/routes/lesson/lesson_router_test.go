package lesson_test

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

	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	lessonQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/lesson"
)

type mockLessonRepository struct {
	lessons []entities.Lesson
	items   []entities.LessonItem
}

func (m *mockLessonRepository) FindByIDOrSlug(ctx context.Context, identifier string) (*entities.Lesson, error) {
	for _, l := range m.lessons {
		if l.ID == identifier || l.Slug == identifier {
			return &l, nil
		}
	}
	return nil, errors.New("lesson not found")
}

func (m *mockLessonRepository) FindByCourseID(ctx context.Context, courseID string) ([]entities.Lesson, error) {
	var res []entities.Lesson
	for _, l := range m.lessons {
		if l.CourseID == courseID {
			res = append(res, l)
		}
	}
	return res, nil
}

func (m *mockLessonRepository) FindByID(ctx context.Context, id string) (*entities.Lesson, error) {
	for _, l := range m.lessons {
		if l.ID == id {
			return &l, nil
		}
	}
	return nil, errors.New("lesson not found")
}

func (m *mockLessonRepository) Create(ctx context.Context, l *entities.Lesson) error {
	m.lessons = append(m.lessons, *l)
	return nil
}

func (m *mockLessonRepository) Update(ctx context.Context, l *entities.Lesson) error {
	for i, existing := range m.lessons {
		if existing.ID == l.ID {
			m.lessons[i] = *l
			return nil
		}
	}
	return errors.New("lesson not found")
}

func (m *mockLessonRepository) Delete(ctx context.Context, id string) error {
	var remaining []entities.Lesson
	for _, l := range m.lessons {
		if l.ID != id {
			remaining = append(remaining, l)
		}
	}
	m.lessons = remaining
	return nil
}

func (m *mockLessonRepository) SaveItem(ctx context.Context, item *entities.LessonItem) error {
	for i, existing := range m.items {
		if existing.ID == item.ID {
			m.items[i] = *item
			return nil
		}
	}
	m.items = append(m.items, *item)
	return nil
}

func (m *mockLessonRepository) DeleteItem(ctx context.Context, itemID string) error {
	var remaining []entities.LessonItem
	for _, it := range m.items {
		if it.ID != itemID {
			remaining = append(remaining, it)
		}
	}
	m.items = remaining
	return nil
}

type mockCourseRepoForLesson struct {
	courses []entities.Course
}

func (m *mockCourseRepoForLesson) Create(ctx context.Context, course *entities.Course) error {
	m.courses = append(m.courses, *course)
	return nil
}

func (m *mockCourseRepoForLesson) FindByAuthor(ctx context.Context, authorID string) ([]entities.Course, error) {
	return nil, nil
}

func (m *mockCourseRepoForLesson) FindPublished(ctx context.Context) ([]entities.Course, error) {
	return nil, nil
}

func (m *mockCourseRepoForLesson) FindByID(ctx context.Context, id string) (*entities.Course, error) {
	for _, c := range m.courses {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, errors.New("course not found")
}

func (m *mockCourseRepoForLesson) Update(ctx context.Context, course *entities.Course) error {
	return nil
}

func setupLessonTestServer() (*echo.Echo, *security.JWTService, *mockLessonRepository, *mockCourseRepoForLesson) {
	e := echo.New()
	jwtSvc := security.NewJWTService("test-secret-key-32-bytes-minimum!", 15*time.Minute, time.Hour)
	lessonRepo := &mockLessonRepository{}
	courseRepo := &mockCourseRepoForLesson{}

	listLessonsHandler := lessonQuery.NewListCourseLessonsHandler(lessonRepo)
	createLessonHandler := lessonCmd.NewCreateLessonHandler(lessonRepo, courseRepo)
	updateLessonHandler := lessonCmd.NewUpdateLessonHandler(lessonRepo, courseRepo)
	deleteLessonHandler := lessonCmd.NewDeleteLessonHandler(lessonRepo, courseRepo)
	saveLessonItemHandler := lessonCmd.NewSaveLessonItemHandler(lessonRepo, courseRepo)
	deleteLessonItemHandler := lessonCmd.NewDeleteLessonItemHandler(lessonRepo, courseRepo)

	router := lesson.NewLessonRouter(
		lessonRepo,
		listLessonsHandler,
		createLessonHandler,
		updateLessonHandler,
		deleteLessonHandler,
		saveLessonItemHandler,
		deleteLessonItemHandler,
	)

	authRequired := restMiddleware.RequireAuth(jwtSvc)
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)

	v1 := e.Group("/api/v1")
	router.RegisterRoutes(v1, authRequired, instructorOrAdmin)

	return e, jwtSvc, lessonRepo, courseRepo
}

func TestLessonRoutesRBAC(t *testing.T) {
	e, jwtSvc, lessonRepo, courseRepo := setupLessonTestServer()

	teacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "teacher-1",
		Email: "teacher1@wordtap.app",
		Role:  entities.RoleInstructor,
	})

	otherTeacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "teacher-2",
		Email: "teacher2@wordtap.app",
		Role:  entities.RoleInstructor,
	})

	studentToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "student-1",
		Email: "student@wordtap.app",
		Role:  entities.RoleStudent,
	})

	authorID := "teacher-1"
	courseRepo.courses = append(courseRepo.courses, entities.Course{
		ID:        "course-100",
		Title:     "Inglés para Viajeros",
		Status:    entities.ContentStatusDraft,
		CreatedBy: &authorID,
	})

	var createdLessonID string

	t.Run("student cannot create lesson (403 Forbidden)", func(t *testing.T) {
		body := `{"title":"Lección 1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/course-100/lessons", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+studentToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for student, got %d", rec.Code)
		}
	})

	t.Run("other teacher cannot create lesson on foreign course (400 Bad Request)", func(t *testing.T) {
		body := `{"title":"Lección 1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/course-100/lessons", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+otherTeacherToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for foreign teacher, got %d", rec.Code)
		}
	})

	t.Run("author can create lesson (201 Created)", func(t *testing.T) {
		body := `{"title":"Lección 1: En el Aeropuerto","description":"Vocabulario básico","estimated_minutes":15,"sort_order":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/course-100/lessons", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var created entities.Lesson
		_ = json.Unmarshal(rec.Body.Bytes(), &created)
		if created.ID == "" || created.Title != "Lección 1: En el Aeropuerto" {
			t.Errorf("unexpected lesson created: %+v", created)
		}
		createdLessonID = created.ID
	})

	t.Run("public can list course lessons (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-100/lessons", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var list []entities.Lesson
		_ = json.Unmarshal(rec.Body.Bytes(), &list)
		if len(list) != 1 {
			t.Errorf("expected 1 lesson, got %d", len(list))
		}
	})

	t.Run("author can update lesson (200 OK)", func(t *testing.T) {
		body := `{"title":"Lección 1: En el Aeropuerto Modificada"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/lessons/"+createdLessonID, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on update, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("author can save lesson card (200 OK)", func(t *testing.T) {
		body := `{"id":"card-1","item_type":"word","content_text":"boarding pass","sort_order":1,"is_required":true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/lessons/"+createdLessonID+"/items", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on card save, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		if len(lessonRepo.items) != 1 {
			t.Fatalf("expected 1 card in repo, got %d", len(lessonRepo.items))
		}
	})

	t.Run("author can delete lesson card (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/lessons/"+createdLessonID+"/items/card-1", nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on card delete, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		if len(lessonRepo.items) != 0 {
			t.Errorf("expected card to be deleted, items: %d", len(lessonRepo.items))
		}
	})

	t.Run("author can delete lesson (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/lessons/"+createdLessonID, nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on lesson delete, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		if len(lessonRepo.lessons) != 0 {
			t.Errorf("expected lesson to be deleted, lessons: %d", len(lessonRepo.lessons))
		}
	})
}
