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

func (m *mockCourseRepoForLesson) FindAll(ctx context.Context) ([]entities.Course, error) {
	return m.courses, nil
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

func (m *mockCourseRepoForLesson) Delete(ctx context.Context, id string) error {
	return nil
}

func setupLessonTestServer() (*echo.Echo, *security.JWTService, *mockLessonRepository, *mockCourseRepoForLesson) {
	e := echo.New()
	jwtSvc := security.NewJWTService("test-secret-key-32-bytes-minimum!", 15*time.Minute, time.Hour)
	lessonRepo := &mockLessonRepository{}
	courseRepo := &mockCourseRepoForLesson{}

	listLessonsHandler := lessonQuery.NewListCourseLessonsHandler(lessonRepo, courseRepo)
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
	studentOrStaff := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin, entities.RoleModerator, entities.RoleStudent)
	instructorOrAdmin := restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin)

	v1 := e.Group("/api/v1")
	router.RegisterRoutes(v1, authRequired, studentOrStaff, instructorOrAdmin)

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

	t.Run("author can list draft course lessons (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-100/lessons", nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
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

	t.Run("anonymous gets 401, student cannot list draft course lessons (403)", func(t *testing.T) {
		reqAnon := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-100/lessons", nil)
		recAnon := httptest.NewRecorder()
		e.ServeHTTP(recAnon, reqAnon)
		if recAnon.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for anonymous, got %d", recAnon.Code)
		}

		reqStudent := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-100/lessons", nil)
		reqStudent.Header.Set("Authorization", "Bearer "+studentToken)
		recStudent := httptest.NewRecorder()
		e.ServeHTTP(recStudent, reqStudent)
		if recStudent.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for student on draft course, got %d", recStudent.Code)
		}
	})

	t.Run("student and staff can list published course lessons (200 OK)", func(t *testing.T) {
		modToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
			ID:    "mod-1",
			Email: "moderator@wordtap.app",
			Role:  entities.RoleModerator,
		})

		pubCourseID := "course-published-200"
		courseRepo.courses = append(courseRepo.courses, entities.Course{
			ID:        pubCourseID,
			Title:     "Curso Publicado",
			Status:    entities.ContentStatusPublished,
			CreatedBy: &authorID,
		})
		lessonRepo.lessons = append(lessonRepo.lessons, entities.Lesson{
			ID:       "lesson-pub-1",
			CourseID: pubCourseID,
			Title:    "Lección Pública",
			Status:   entities.ContentStatusPublished,
		})

		// Student -> 200 OK
		reqStudent := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubCourseID+"/lessons", nil)
		reqStudent.Header.Set("Authorization", "Bearer "+studentToken)
		recStudent := httptest.NewRecorder()
		e.ServeHTTP(recStudent, reqStudent)
		if recStudent.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for student on published course lessons, got %d", recStudent.Code)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubCourseID+"/lessons", nil)
		req.Header.Set("Authorization", "Bearer "+modToken)
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

		for _, l := range lessonRepo.lessons {
			if l.ID == createdLessonID {
				t.Errorf("expected lesson %s to be deleted", createdLessonID)
			}
		}
	})

	t.Run("anonymous access to lessons of draft course returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-100/lessons", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for draft course lessons without auth, got %d", rec.Code)
		}
	})

	t.Run("foreign teacher access to lessons of draft course returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-100/lessons", nil)
		req.Header.Set("Authorization", "Bearer "+otherTeacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for draft course lessons by foreign teacher, got %d", rec.Code)
		}
	})

	t.Run("author access to lessons of draft course returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/course-100/lessons", nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for author, got %d", rec.Code)
		}
	})
}

func TestLessonAdminSupervisionIsReadOnly(t *testing.T) {
	e, jwtSvc, lessonRepo, courseRepo := setupLessonTestServer()

	adminToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "admin-1",
		Email: "admin@wordtap.app",
		Role:  entities.RoleAdmin,
	})
	teacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "teacher-1",
		Email: "teacher1@wordtap.app",
		Role:  entities.RoleInstructor,
	})

	teacherID := "teacher-1"
	adminID := "admin-1"

	courseRepo.courses = append(courseRepo.courses, entities.Course{
		ID:        "course-teacher",
		Title:     "Curso del Docente",
		Status:    entities.ContentStatusDraft,
		OwnerType: entities.OwnerTypeTeacher,
		CreatedBy: &teacherID,
	})
	lessonRepo.lessons = append(lessonRepo.lessons, entities.Lesson{
		ID:       "lesson-teacher",
		CourseID: "course-teacher",
		Title:    "Lección del Docente",
		Status:   entities.ContentStatusDraft,
	})
	lessonRepo.items = append(lessonRepo.items, entities.LessonItem{
		ID:       "item-teacher",
		LessonID: "lesson-teacher",
		ItemType: entities.LessonContentTypeWord,
	})

	courseRepo.courses = append(courseRepo.courses, entities.Course{
		ID:        "course-platform",
		Title:     "Curso de Wordtap",
		Status:    entities.ContentStatusDraft,
		OwnerType: entities.OwnerTypePlatform,
		CreatedBy: &adminID,
	})
	lessonRepo.lessons = append(lessonRepo.lessons, entities.Lesson{
		ID:       "lesson-platform",
		CourseID: "course-platform",
		Title:    "Lección de Wordtap",
		Status:   entities.ContentStatusDraft,
	})

	do := func(method, path, body, token string) *httptest.ResponseRecorder {
		var requestBody *bytes.Buffer
		if body == "" {
			requestBody = bytes.NewBuffer(nil)
		} else {
			requestBody = bytes.NewBufferString(body)
		}
		req := httptest.NewRequest(method, path, requestBody)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	t.Run("admin cannot create a lesson in a teacher course", func(t *testing.T) {
		rec := do(http.MethodPost, "/api/v1/courses/course-teacher/lessons", `{"title":"Lección Admin"}`, adminToken)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin cannot update a teacher lesson", func(t *testing.T) {
		rec := do(http.MethodPut, "/api/v1/lessons/lesson-teacher", `{"title":"Editada por admin"}`, adminToken)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin cannot save or delete teacher lesson cards", func(t *testing.T) {
		saveRec := do(http.MethodPost, "/api/v1/lessons/lesson-teacher/items", `{"id":"item-admin","item_type":"word","content_text":"x","sort_order":1}`, adminToken)
		if saveRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 saving card, got %d. Body: %s", saveRec.Code, saveRec.Body.String())
		}

		deleteRec := do(http.MethodDelete, "/api/v1/lessons/lesson-teacher/items/item-teacher", "", adminToken)
		if deleteRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 deleting card, got %d. Body: %s", deleteRec.Code, deleteRec.Body.String())
		}
	})

	t.Run("admin cannot delete a teacher lesson", func(t *testing.T) {
		rec := do(http.MethodDelete, "/api/v1/lessons/lesson-teacher", "", adminToken)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin can manage lessons of a platform course", func(t *testing.T) {
		createRec := do(http.MethodPost, "/api/v1/courses/course-platform/lessons", `{"title":"Nueva Lección de Wordtap"}`, adminToken)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("expected 201 creating platform lesson, got %d. Body: %s", createRec.Code, createRec.Body.String())
		}

		updateRec := do(http.MethodPut, "/api/v1/lessons/lesson-platform", `{"title":"Lección de Wordtap Actualizada"}`, adminToken)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("expected 200 updating platform lesson, got %d. Body: %s", updateRec.Code, updateRec.Body.String())
		}

		saveRec := do(http.MethodPost, "/api/v1/lessons/lesson-platform/items", `{"id":"item-platform","item_type":"word","content_text":"x","sort_order":1}`, adminToken)
		if saveRec.Code != http.StatusOK {
			t.Fatalf("expected 200 saving platform card, got %d. Body: %s", saveRec.Code, saveRec.Body.String())
		}

		deleteItemRec := do(http.MethodDelete, "/api/v1/lessons/lesson-platform/items/item-platform", "", adminToken)
		if deleteItemRec.Code != http.StatusOK {
			t.Fatalf("expected 200 deleting platform card, got %d. Body: %s", deleteItemRec.Code, deleteItemRec.Body.String())
		}

		deleteLessonRec := do(http.MethodDelete, "/api/v1/lessons/lesson-platform", "", adminToken)
		if deleteLessonRec.Code != http.StatusOK {
			t.Fatalf("expected 200 deleting platform lesson, got %d. Body: %s", deleteLessonRec.Code, deleteLessonRec.Body.String())
		}
	})

	// The teacher still owns their own course.
	t.Run("teacher can still update their own lesson", func(t *testing.T) {
		rec := do(http.MethodPut, "/api/v1/lessons/lesson-teacher", `{"title":"Lección del Docente Actualizada"}`, teacherToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}
