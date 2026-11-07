package classroom_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	classroomCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	classroomQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/classroom"
)

func setupStudentClassroomRouter() (*echo.Echo, *mockClassroomRepo) {
	e := echo.New()
	repo := newMockClassroomRepo()

	enrollHandler := classroomCmd.NewEnrollStudentHandler(repo)
	listStudentHandler := classroomQuery.NewListStudentClassroomsHandler(repo)

	r := classroom.NewStudentClassroomRouter(listStudentHandler, enrollHandler)

	v1 := e.Group("/api/v1")
	passMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "student-uuid-99")
			return next(c)
		}
	}
	r.RegisterRoutes(v1, passMiddleware)

	return e, repo
}

func TestStudentClassroomOnlyDraftCourses(t *testing.T) {
	e, repo := setupStudentClassroomRouter()
	repo.classrooms = []entities.Classroom{{
		ID: "classroom", ResolvedCourses: []entities.Course{{ID: "draft", Status: entities.ContentStatusDraft}},
	}}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/student/classrooms", nil))
	var resp struct {
		Data []struct {
			AssignedCourses []any `json:"assignedCourses"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || rec.Code != http.StatusOK || len(resp.Data) != 1 || resp.Data[0].AssignedCourses == nil || len(resp.Data[0].AssignedCourses) != 0 {
		t.Fatalf("expected an empty courses array, status %d body %s error %v", rec.Code, rec.Body.String(), err)
	}
}

func TestStudentClassroomRouter(t *testing.T) {
	e, repo := setupStudentClassroomRouter()

	// Crear aula previa
	testClassroom := &entities.Classroom{
		ID:              "cls-student-1",
		Name:            "Inglés B1 Mañana",
		Code:            "WT-4B7K",
		Level:           "B1",
		WeeklyGoalWords: 50,
		IsActive:        true,
		ResolvedCourses: []entities.Course{{ID: "course-1", Title: "Real course one", Status: entities.ContentStatusPublished, AccessTier: entities.AccessTierCourse, PriceCents: 2750, Lessons: []entities.Lesson{{ID: "lesson-1", Status: entities.ContentStatusPublished}}}, {ID: "course-2", Title: "Real course two", Status: entities.ContentStatusDraft}},
	}
	_ = repo.Create(context.Background(), testClassroom)

	t.Run("POST /api/v1/classrooms/join une al estudiante exitosamente", func(t *testing.T) {
		payload := map[string]any{"code": "WT-4B7K"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/classrooms/join", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código esperado %d, obtenido %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /api/v1/student/classrooms devuelve listado formateado", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/student/classrooms", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código esperado %d, obtenido %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool             `json:"success"`
			Data    []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("fallo al decodificar JSON: %v", err)
		}

		if !resp.Success {
			t.Fatalf("se esperaba success = true")
		}
		if len(resp.Data) != 1 {
			t.Fatalf("se esperaba 1 aula, obtenidas %d", len(resp.Data))
		}
		if resp.Data[0]["code"] != "WT-4B7K" {
			t.Errorf("código inesperado: %v", resp.Data[0]["code"])
		}
		courses, ok := resp.Data[0]["assignedCourses"].([]any)
		if !ok || len(courses) != 1 {
			t.Fatalf("expected only published courses, got %v", resp.Data[0]["assignedCourses"])
		}
		if courses[0].(map[string]any)["title"] != "Real course one" {
			t.Errorf("published course title not returned: %v", courses[0])
		}
		if courses[0].(map[string]any)["available"] != true || courses[0].(map[string]any)["lessonCount"] != float64(1) {
			t.Errorf("course availability or lesson count incorrect: %v", courses)
		}
		if courses[0].(map[string]any)["access_tier"] != string(entities.AccessTierCourse) || courses[0].(map[string]any)["price_cents"] != float64(2750) {
			t.Errorf("course pricing incorrect: %v", courses)
		}
		games, ok := resp.Data[0]["activeGames"].([]any)
		if !ok || len(games) != 0 {
			t.Errorf("un aula sin juegos configurados no debe mostrar juegos predeterminados: %v", resp.Data[0]["activeGames"])
		}
		teacher, ok := resp.Data[0]["teacher"].(map[string]any)
		if !ok || teacher["credential"] != nil {
			t.Errorf("la respuesta no debe inventar credenciales docentes: %v", resp.Data[0]["teacher"])
		}
	})
}
