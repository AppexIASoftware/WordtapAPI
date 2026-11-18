package course_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/course"
	"github.com/labstack/echo/v5"
)

type stubCoursePurchaseReader struct {
	purchased bool
	err       error
}

func (s stubCoursePurchaseReader) HasPurchase(_ context.Context, userID, courseID string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.purchased, nil
}

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

		// Student -> 200 OK (acceso a cursos publicados)
		reqStudent := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
		reqStudent.Header.Set("Authorization", "Bearer "+studentToken)
		recStudent := httptest.NewRecorder()
		e.ServeHTTP(recStudent, reqStudent)
		if recStudent.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for student, got %d", recStudent.Code)
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

	t.Run("anonymous rejected (401), student gets published course (200 OK) but draft is forbidden (403)", func(t *testing.T) {
		reqAnon := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubCourseID, nil)
		recAnon := httptest.NewRecorder()
		e.ServeHTTP(recAnon, reqAnon)
		if recAnon.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for anonymous detail, got %d", recAnon.Code)
		}

		// Student on published course -> 200 OK
		reqStudent := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubCourseID, nil)
		reqStudent.Header.Set("Authorization", "Bearer "+studentToken)
		recStudent := httptest.NewRecorder()
		e.ServeHTTP(recStudent, reqStudent)
		if recStudent.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for student on published course, got %d", recStudent.Code)
		}

		// Student on draft course -> 403 Forbidden
		reqDraft := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+draftCourseID, nil)
		reqDraft.Header.Set("Authorization", "Bearer "+studentToken)
		recDraft := httptest.NewRecorder()
		e.ServeHTTP(recDraft, reqDraft)
		if recDraft.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for student on draft course, got %d", recDraft.Code)
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

func TestCourseDetail_ContentPolicyHidesUnpurchasedTeacherLessons(t *testing.T) {
	e := echo.New()
	jwtSvc := security.NewJWTService("test-secret-key-32-bytes-minimum!", 15*time.Minute, time.Hour)
	repo := &mockCourseRepository{}
	reviewRepo := &mockCourseReviewRepository{courses: repo}
	courseRouter := course.NewCourseRouter(
		courseCmd.NewCreateCourseHandler(repo),
		courseCmd.NewUpdateCourseHandler(repo, reviewRepo),
		courseCmd.NewDeleteCourseHandler(repo),
		courseQuery.NewListTeacherCoursesHandler(repo),
		courseQuery.NewListPublishedCoursesHandler(repo),
		courseQuery.NewListAdminCoursesHandler(repo),
		courseQuery.NewGetCourseDetailHandler(repo),
	)
	authRequired := restMiddleware.RequireAuth(jwtSvc)
	pass := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	courseRouter.RegisterRoutes(e.Group("/api/v1"), authRequired, pass, pass, pass, pass, pass)

	authorID := "teacher-owner-9"
	teacherCourseID := "teacher-paid-course-1"
	platformCourseID := "platform-course-1"

	repo.courses = append(repo.courses,
		entities.Course{
			ID:        teacherCourseID,
			Title:     "Curso Pagado Docente",
			OwnerType: entities.OwnerTypeTeacher,
			Status:    entities.ContentStatusPublished,
			CreatedBy: &authorID,
			Lessons: []entities.Lesson{
				{ID: "l-teacher-1", CourseID: teacherCourseID, Title: "Unidad 1", Status: entities.ContentStatusPublished},
			},
		},
		entities.Course{
			ID:        platformCourseID,
			Title:     "Curso Oficial Plataforma",
			OwnerType: entities.OwnerTypePlatform,
			Status:    entities.ContentStatusPublished,
			Lessons: []entities.Lesson{
				{ID: "l-platform-1", CourseID: platformCourseID, Title: "Oficial 1", Status: entities.ContentStatusPublished},
			},
		},
	)

	studentToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
		ID:    "student-buyer-1",
		Email: "buyer@wordtap.app",
		Role:  entities.RoleStudent,
	})

	callDetail := func(courseID string) (*httptest.ResponseRecorder, course.PublicCourseDTO) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseID, nil)
		req.Header.Set("Authorization", "Bearer "+studentToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		var dto course.PublicCourseDTO
		_ = json.Unmarshal(rec.Body.Bytes(), &dto)
		return rec, dto
	}

	t.Run("unpurchased student gets 200 metadata with LessonsCount=1 and Lessons withheld", func(t *testing.T) {
		courseRouter.WithContentPolicy(access.NewContentPolicy(stubCoursePurchaseReader{purchased: false}, true))
		rec, dto := callDetail(teacherCourseID)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if dto.LessonsCount != 1 || len(dto.Lessons) != 0 {
			t.Fatalf("expected LessonsCount=1 and withheld lessons for unpurchased student, got count=%d lessons=%d", dto.LessonsCount, len(dto.Lessons))
		}
	})

	t.Run("purchased student and platform course receive full published lessons", func(t *testing.T) {
		courseRouter.WithContentPolicy(access.NewContentPolicy(stubCoursePurchaseReader{purchased: true}, true))
		recPaid, dtoPaid := callDetail(teacherCourseID)
		if recPaid.Code != http.StatusOK || len(dtoPaid.Lessons) != 1 {
			t.Fatalf("expected 200 with 1 lesson for purchaser, got code=%d lessons=%d", recPaid.Code, len(dtoPaid.Lessons))
		}

		courseRouter.WithContentPolicy(access.NewContentPolicy(stubCoursePurchaseReader{purchased: false}, true))
		recPlat, dtoPlat := callDetail(platformCourseID)
		if recPlat.Code != http.StatusOK || len(dtoPlat.Lessons) != 1 {
			t.Fatalf("expected 200 with 1 lesson for platform course, got code=%d lessons=%d", recPlat.Code, len(dtoPlat.Lessons))
		}
	})

	t.Run("unexpected purchase store error returns 500", func(t *testing.T) {
		courseRouter.WithContentPolicy(access.NewContentPolicy(stubCoursePurchaseReader{err: errors.New("db down")}, true))
		rec, _ := callDetail(teacherCourseID)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 on unexpected storage error, got %d", rec.Code)
		}
	})

	t.Run("competitor instructor gets 403 on another teacher published course", func(t *testing.T) {
		courseRouter.WithContentPolicy(access.NewContentPolicy(stubCoursePurchaseReader{purchased: false}, true))
		otherTeacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
			ID: "teacher-competitor-77", Role: entities.RoleInstructor,
		})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+teacherCourseID, nil)
		req.Header.Set("Authorization", "Bearer "+otherTeacherToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for competitor instructor, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("moderator audit=1 cannot expose draft lessons inside published teacher course", func(t *testing.T) {
		pubWithDraft := "teacher-pub-with-draft-1"
		repo.courses = append(repo.courses, entities.Course{
			ID:        pubWithDraft,
			Title:     "Publicado Docente con Borrador",
			OwnerType: entities.OwnerTypeTeacher,
			Status:    entities.ContentStatusPublished,
			CreatedBy: &authorID,
			Lessons: []entities.Lesson{
				{ID: "l-pub-ok", CourseID: pubWithDraft, Title: "Pública", Status: entities.ContentStatusPublished},
				{ID: "l-priv-draft", CourseID: pubWithDraft, Title: "Borrador Privado", Status: entities.ContentStatusDraft,
					Items: []entities.LessonItem{{ID: "secret-card", LessonID: "l-priv-draft"}}},
			},
		})
		modToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{
			ID: "mod-audit-1", Role: entities.RoleModerator,
		})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+pubWithDraft+"?audit=1", nil)
		req.Header.Set("Authorization", "Bearer "+modToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for moderator, got %d", rec.Code)
		}
		var dto course.PublicCourseDTO
		_ = json.Unmarshal(rec.Body.Bytes(), &dto)
		if len(dto.Lessons) != 1 || dto.Lessons[0].ID != "l-pub-ok" {
			t.Fatalf("moderator ?audit=1 leaked draft lesson on published teacher course: %+v", dto.Lessons)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}
