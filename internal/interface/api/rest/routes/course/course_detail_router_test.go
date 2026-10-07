package course_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/course"
)

func TestCourseDetailRoutes(t *testing.T) {
	e, jwtSvc, repo, _ := setupTestServer()

	teacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "teacher-owner-1",
		Email:      "owner@wordtap.app",
		Role:       entities.RoleInstructor,
		AccessTier: entities.AccessTierCourse,
	})

	otherTeacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:         "teacher-other-2",
		Email:      "other@wordtap.app",
		Role:       entities.RoleInstructor,
		AccessTier: entities.AccessTierCourse,
	})

	adminToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "admin-user-1",
		Email: "admin@wordtap.app",
		Role:  entities.RoleAdmin,
	})

	studentToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "student-user-1",
		Email: "student@wordtap.app",
		Role:  entities.RoleStudent,
	})

	modToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "mod-user-1",
		Email: "moderator@wordtap.app",
		Role:  entities.RoleModerator,
	})

	authorID := "teacher-owner-1"
	draftCourseID := "course-draft-detail-1"
	pubCourseID := "course-published-detail-1"

	repo.courses = append(repo.courses,
		entities.Course{
			ID:        draftCourseID,
			Title:     "Draft Course for Detail Test",
			Status:    entities.ContentStatusDraft,
			CreatedBy: &authorID,
		},
		entities.Course{
			ID:        pubCourseID,
			Title:     "Published Course for Detail Test",
			Status:    entities.ContentStatusPublished,
			CreatedBy: &authorID,
			Lessons: []entities.Lesson{
				{
					ID:       "lesson-pub-1",
					CourseID: pubCourseID,
					Title:    "Published Lesson",
					Status:   entities.ContentStatusPublished,
					Items: []entities.LessonItem{
						{
							ID:          "item-private-1",
							LessonID:    "lesson-pub-1",
							ContentText: stringPtr("gated secret item"),
						},
					},
				},
				{
					ID:       "lesson-draft-2",
					CourseID: pubCourseID,
					Title:    "Draft Lesson inside Published Course",
					Status:   entities.ContentStatusDraft,
				},
			},
		},
	)

	t.Run("catalog RBAC: rejects anonymous (401) and student (403), allows staff (200 OK)", func(t *testing.T) {
		// Anonymous -> 401
		reqAnon := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
		recAnon := httptest.NewRecorder()
		e.ServeHTTP(recAnon, reqAnon)
		if recAnon.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for anonymous, got %d", recAnon.Code)
		}

		// Student -> 403
		reqStudent := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
		reqStudent.Header.Set("Authorization", "Bearer "+studentToken)
		recStudent := httptest.NewRecorder()
		e.ServeHTTP(recStudent, reqStudent)
		if recStudent.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for student, got %d", recStudent.Code)
		}

		// Moderator -> 200 OK
		reqMod := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
		reqMod.Header.Set("Authorization", "Bearer "+modToken)
		recMod := httptest.NewRecorder()
		e.ServeHTTP(recMod, reqMod)
		if recMod.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for moderator, got %d", recMod.Code)
		}

		var list []course.PublicCourseDTO
		if err := json.Unmarshal(recMod.Body.Bytes(), &list); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		var rawList []map[string]any
		_ = json.Unmarshal(recMod.Body.Bytes(), &rawList)
		for _, m := range rawList {
			if _, exists := m["created_by"]; exists {
				t.Errorf("expected 'created_by' to be omitted in catalog")
			}
		}
	})

	t.Run("anonymous and student cannot get course detail (401 / 403)", func(t *testing.T) {
		reqAnon := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubCourseID, nil)
		recAnon := httptest.NewRecorder()
		e.ServeHTTP(recAnon, reqAnon)
		if recAnon.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for anonymous detail, got %d", recAnon.Code)
		}

		reqStudent := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubCourseID, nil)
		reqStudent.Header.Set("Authorization", "Bearer "+studentToken)
		recStudent := httptest.NewRecorder()
		e.ServeHTTP(recStudent, reqStudent)
		if recStudent.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for student detail, got %d", recStudent.Code)
		}
	})

	t.Run("foreign teacher cannot get draft course detail (403 Forbidden)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+draftCourseID, nil)
		req.Header.Set("Authorization", "Bearer "+otherTeacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for draft course accessed by foreign teacher, got %d", rec.Code)
		}
	})

	t.Run("author can get own draft course detail (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+draftCourseID, nil)
		req.Header.Set("Authorization", "Bearer "+teacherToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for author on draft course, got %d", rec.Code)
		}
	})

	t.Run("admin can get draft course detail (200 OK)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+draftCourseID, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for admin on draft course, got %d", rec.Code)
		}
	})

	t.Run("staff (moderator) can get published course detail (200 OK) without private data or gated items", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubCourseID, nil)
		req.Header.Set("Authorization", "Bearer "+modToken)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for published course, got %d", rec.Code)
		}

		var pubCourse course.PublicCourseDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &pubCourse); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if pubCourse.ID != pubCourseID {
			t.Errorf("expected ID '%s', got '%s'", pubCourseID, pubCourse.ID)
		}

		// Only published lesson should be present (1 lesson, not 2)
		if len(pubCourse.Lessons) != 1 {
			t.Errorf("expected 1 published lesson, got %d", len(pubCourse.Lessons))
		}

		var rawMap map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &rawMap)
		if _, exists := rawMap["created_by"]; exists {
			t.Errorf("expected 'created_by' to be omitted in public course detail")
		}

		// Verify lessons do not serialize items
		lessonsRaw, ok := rawMap["lessons"].([]any)
		if !ok || len(lessonsRaw) == 0 {
			t.Fatalf("expected lessons array in public response")
		}
		firstLesson := lessonsRaw[0].(map[string]any)
		if _, exists := firstLesson["items"]; exists {
			t.Errorf("expected gated 'items' to be omitted in public lesson projection")
		}
	})

	t.Run("dedicated teacher private detail path GET /api/v1/teacher/courses/:id", func(t *testing.T) {
		// 1. Unauthenticated -> 401
		reqUnauth := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/courses/"+draftCourseID, nil)
		recUnauth := httptest.NewRecorder()
		e.ServeHTTP(recUnauth, reqUnauth)
		if recUnauth.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", recUnauth.Code)
		}

		// 2. Foreign teacher -> 403 Forbidden
		reqForeign := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/courses/"+draftCourseID, nil)
		reqForeign.Header.Set("Authorization", "Bearer "+otherTeacherToken)
		recForeign := httptest.NewRecorder()
		e.ServeHTTP(recForeign, reqForeign)
		if recForeign.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for foreign teacher, got %d", recForeign.Code)
		}

		// 3. Author teacher -> 200 OK with full private course detail
		reqAuthor := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/courses/"+draftCourseID, nil)
		reqAuthor.Header.Set("Authorization", "Bearer "+teacherToken)
		recAuthor := httptest.NewRecorder()
		e.ServeHTTP(recAuthor, reqAuthor)
		if recAuthor.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for author teacher, got %d", recAuthor.Code)
		}

		// 4. Admin -> 200 OK
		reqAdmin := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/courses/"+draftCourseID, nil)
		reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
		recAdmin := httptest.NewRecorder()
		e.ServeHTTP(recAdmin, reqAdmin)
		if recAdmin.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for admin, got %d", recAdmin.Code)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}
