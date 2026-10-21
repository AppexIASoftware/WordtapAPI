package course_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestAdminCannotEditOrDeleteTeacherCourse(t *testing.T) {
	e, jwtSvc, _, _ := setupTestServer()

	teacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "teacher-123",
		Email:      "teacher@wordtap.app",
		Role:       entities.RoleInstructor,
		AccessTier: entities.AccessTierCourse,
	})
	adminToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "admin-root",
		Email:      "admin@wordtap.app",
		Role:       entities.RoleAdmin,
		AccessTier: entities.AccessTierCourse,
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBufferString(`{"title":"Curso de Docente para Supervisión","level":"A2"}`))
	createReq.Header.Set("Authorization", "Bearer "+teacherToken)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating teacher course, got %d. Body: %s", createRec.Code, createRec.Body.String())
	}

	var teacherCourse entities.Course
	if err := json.Unmarshal(createRec.Body.Bytes(), &teacherCourse); err != nil {
		t.Fatalf("failed to decode teacher course: %v", err)
	}
	if teacherCourse.OwnerType != entities.OwnerTypeTeacher {
		t.Fatalf("expected teacher ownership, got '%s'", teacherCourse.OwnerType)
	}

	editReq := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+teacherCourse.ID, bytes.NewBufferString(`{"title":"Editado por admin"}`))
	editReq.Header.Set("Authorization", "Bearer "+adminToken)
	editReq.Header.Set("Content-Type", "application/json")
	editRec := httptest.NewRecorder()
	e.ServeHTTP(editRec, editReq)
	if editRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when an admin edits teacher course content, got %d", editRec.Code)
	}

	priceReq := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+teacherCourse.ID, bytes.NewBufferString(`{"access_tier":"course","price_cents":5000}`))
	priceReq.Header.Set("Authorization", "Bearer "+adminToken)
	priceReq.Header.Set("Content-Type", "application/json")
	priceRec := httptest.NewRecorder()
	e.ServeHTTP(priceRec, priceReq)
	if priceRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when an admin changes teacher course pricing, got %d", priceRec.Code)
	}

	statusReq := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+teacherCourse.ID, bytes.NewBufferString(`{"status":"published"}`))
	statusReq.Header.Set("Authorization", "Bearer "+adminToken)
	statusReq.Header.Set("Content-Type", "application/json")
	statusRec := httptest.NewRecorder()
	e.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected 200 when an admin publishes a teacher course, got %d. Body: %s", statusRec.Code, statusRec.Body.String())
	}

	var published entities.Course
	_ = json.Unmarshal(statusRec.Body.Bytes(), &published)
	if published.Status != entities.ContentStatusPublished {
		t.Errorf("expected published status, got '%s'", published.Status)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+teacherCourse.ID, nil)
	deleteReq.Header.Set("Authorization", "Bearer "+adminToken)
	deleteRec := httptest.NewRecorder()
	e.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when an admin deletes a teacher course, got %d", deleteRec.Code)
	}
}
