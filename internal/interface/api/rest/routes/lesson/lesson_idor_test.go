package lesson_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

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

	t.Run("teacher can still update their own lesson", func(t *testing.T) {
		rec := do(http.MethodPut, "/api/v1/lessons/lesson-teacher", `{"title":"Lección del Docente Actualizada"}`, teacherToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestLessonIDORAndReviewGuards(t *testing.T) {
	e, jwtSvc, lessonRepo, courseRepo := setupLessonTestServer()

	t1Token, _, _ := jwtSvc.GenerateAccessToken(&entities.User{ID: "teacher-1", Email: "t1@wordtap.app", Role: entities.RoleInstructor})
	adminToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{ID: "admin-1", Email: "admin@wordtap.app", Role: entities.RoleAdmin})

	t1ID := "teacher-1"
	t2ID := "teacher-2"
	adminID := "admin-1"
	origText := "tarjeta original docente 2"

	courseRepo.courses = append(courseRepo.courses,
		entities.Course{ID: "course-t1-pub", Title: "Publicado T1", Status: entities.ContentStatusPublished, OwnerType: entities.OwnerTypeTeacher, CreatedBy: &t1ID},
		entities.Course{ID: "course-t1-rev", Title: "En Revision T1", Status: entities.ContentStatusInReview, OwnerType: entities.OwnerTypeTeacher, CreatedBy: &t1ID},
		entities.Course{ID: "course-t2", Title: "Curso T2", Status: entities.ContentStatusDraft, OwnerType: entities.OwnerTypeTeacher, CreatedBy: &t2ID},
		entities.Course{ID: "course-plat-pub", Title: "Oficial", Status: entities.ContentStatusPublished, OwnerType: entities.OwnerTypePlatform, CreatedBy: &adminID},
	)
	lessonRepo.lessons = append(lessonRepo.lessons,
		entities.Lesson{ID: "lesson-t1", CourseID: "course-t1-pub", Title: "L1", Status: entities.ContentStatusPublished},
		entities.Lesson{ID: "lesson-t1-rev", CourseID: "course-t1-rev", Title: "L1 Rev", Status: entities.ContentStatusDraft},
		entities.Lesson{ID: "lesson-t2", CourseID: "course-t2", Title: "L2", Status: entities.ContentStatusDraft},
	)
	lessonRepo.items = append(lessonRepo.items, entities.LessonItem{
		ID:          "item-teacher-2",
		LessonID:    "lesson-t2",
		ItemType:    entities.LessonContentTypeWord,
		ContentText: &origText,
	})

	doReq := func(method, path, body, token string) *httptest.ResponseRecorder {
		var buf *bytes.Buffer
		if body == "" {
			buf = bytes.NewBuffer(nil)
		} else {
			buf = bytes.NewBufferString(body)
		}
		req := httptest.NewRequest(method, path, buf)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	t.Run("teacher cannot overwrite or delete foreign lesson card via own lesson endpoint", func(t *testing.T) {
		saveRec := doReq(http.MethodPost, "/api/v1/lessons/lesson-t1/items", `{"id":"item-teacher-2","item_type":"word","content_text":"pisado","sort_order":1}`, t1Token)
		if saveRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 overwriting foreign item, got %d: %s", saveRec.Code, saveRec.Body.String())
		}

		delRec := doReq(http.MethodDelete, "/api/v1/lessons/lesson-t1/items/item-teacher-2", "", t1Token)
		if delRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 deleting foreign item, got %d: %s", delRec.Code, delRec.Body.String())
		}

		if len(lessonRepo.items) != 1 || lessonRepo.items[0].LessonID != "lesson-t2" || *lessonRepo.items[0].ContentText != origText {
			t.Fatalf("teacher-2 item must remain intact, got %+v", lessonRepo.items)
		}
	})

	t.Run("teacher lesson in published teacher course starts in draft while platform course publishes", func(t *testing.T) {
		tRec := doReq(http.MethodPost, "/api/v1/courses/course-t1-pub/lessons", `{"title":"Nueva T1"}`, t1Token)
		if tRec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", tRec.Code, tRec.Body.String())
		}
		var tLesson entities.Lesson
		_ = json.Unmarshal(tRec.Body.Bytes(), &tLesson)
		if tLesson.Status != entities.ContentStatusDraft {
			t.Fatalf("expected teacher lesson to stay draft, got %s", tLesson.Status)
		}

		aRec := doReq(http.MethodPost, "/api/v1/courses/course-plat-pub/lessons", `{"title":"Oficial"}`, adminToken)
		if aRec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", aRec.Code, aRec.Body.String())
		}
		var aLesson entities.Lesson
		_ = json.Unmarshal(aRec.Body.Bytes(), &aLesson)
		if aLesson.Status != entities.ContentStatusPublished {
			t.Fatalf("expected platform lesson to be published, got %s", aLesson.Status)
		}
	})

	t.Run("teacher cannot mutate lessons or cards while course is in_review", func(t *testing.T) {
		if rec := doReq(http.MethodPost, "/api/v1/courses/course-t1-rev/lessons", `{"title":"No"}`, t1Token); rec.Code != http.StatusBadRequest {
			t.Fatalf("create in_review: expected 400, got %d", rec.Code)
		}
		if rec := doReq(http.MethodPut, "/api/v1/lessons/lesson-t1-rev", `{"title":"No"}`, t1Token); rec.Code != http.StatusBadRequest {
			t.Fatalf("update in_review: expected 400, got %d", rec.Code)
		}
		if rec := doReq(http.MethodPost, "/api/v1/lessons/lesson-t1-rev/items", `{"item_type":"word","content_text":"No"}`, t1Token); rec.Code != http.StatusBadRequest {
			t.Fatalf("save card in_review: expected 400, got %d", rec.Code)
		}
		if rec := doReq(http.MethodDelete, "/api/v1/lessons/lesson-t1-rev", "", t1Token); rec.Code != http.StatusBadRequest {
			t.Fatalf("delete in_review: expected 400, got %d", rec.Code)
		}
	})

	_ = context.Background()
}
