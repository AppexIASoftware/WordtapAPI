package classroom_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	classroomCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	classroomQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/classroom"
)

type stubClassroomPurchaseReader struct {
	purchased bool
	calls     int
}

func (s *stubClassroomPurchaseReader) HasPurchase(_ context.Context, _, _ string) (bool, error) {
	s.calls++
	return s.purchased, nil
}

func setupStudentClassroomRouterWithPolicy(reader *stubClassroomPurchaseReader) (*echo.Echo, *mockClassroomRepo) {
	e := echo.New()
	repo := newMockClassroomRepo()

	enrollHandler := classroomCmd.NewEnrollStudentHandler(repo)
	listStudentHandler := classroomQuery.NewListStudentClassroomsHandler(repo)

	r := classroom.NewStudentClassroomRouter(listStudentHandler, enrollHandler).
		WithContentPolicy(access.NewContentPolicy(reader, true))

	v1 := e.Group("/api/v1")
	passMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "student-uuid-99")
			role := c.Request().Header.Get("X-Test-Role")
			if role == "" {
				role = "student"
			}
			c.Set("user_role", role)
			return next(c)
		}
	}
	r.RegisterRoutes(v1, passMiddleware)

	return e, repo
}

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
			role := c.Request().Header.Get("X-Test-Role")
			if role == "" {
				role = "student"
			}
			c.Set("user_role", role)
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

	t.Run("GET /api/v1/student/classrooms reflects purchase policy in has_access while keeping available=true", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			purchased bool
			want      bool
		}{
			{"unpurchased teacher course", false, false},
			{"purchased teacher course", true, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				reader := &stubClassroomPurchaseReader{purchased: tc.purchased}
				ce, crepo := setupStudentClassroomRouterWithPolicy(reader)
				sharedPaid := entities.Course{ID: "course-paid", Title: "Curso Pago", OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished}
				draftCourse := entities.Course{ID: "course-draft", Title: "Borrador", OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusDraft}
				crepo.classrooms = []entities.Classroom{
					{ID: "cls-policy-1", IsActive: true, ResolvedCourses: []entities.Course{sharedPaid, draftCourse}},
					{ID: "cls-policy-2", IsActive: true, ResolvedCourses: []entities.Course{sharedPaid}},
				}
				req := httptest.NewRequest(http.MethodGet, "/api/v1/student/classrooms", nil)
				rec := httptest.NewRecorder()
				ce.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
				}
				if reader.calls != 1 {
					t.Fatalf("expected HasPurchase memoized to 1 call across classrooms (skipping drafts), got %d", reader.calls)
				}
				var out struct {
					Data []struct {
						AssignedCourses []map[string]any `json:"assignedCourses"`
					} `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || len(out.Data) != 2 || len(out.Data[0].AssignedCourses) != 1 || len(out.Data[1].AssignedCourses) != 1 {
					t.Fatalf("unexpected payload: %v %s", err, rec.Body.String())
				}
				entry := out.Data[0].AssignedCourses[0]
				if entry["available"] != true || entry["has_access"] != tc.want {
					t.Fatalf("got available=%v has_access=%v, want available=true has_access=%v", entry["available"], entry["has_access"], tc.want)
				}
			})
		}
	})

	t.Run("non-student roles receive 403 Forbidden on student classroom routes", func(t *testing.T) {
		for _, role := range []string{"instructor", "moderator"} {
			joinReq := httptest.NewRequest(http.MethodPost, "/api/v1/classrooms/join", bytes.NewBufferString(`{"code":"WT-4B7K"}`))
			joinReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			joinReq.Header.Set("X-Test-Role", role)
			joinRec := httptest.NewRecorder()
			e.ServeHTTP(joinRec, joinReq)
			if joinRec.Code != http.StatusForbidden {
				t.Fatalf("expected 403 on join for %s, got %d", role, joinRec.Code)
			}

			listReq := httptest.NewRequest(http.MethodGet, "/api/v1/student/classrooms", nil)
			listReq.Header.Set("X-Test-Role", role)
			listRec := httptest.NewRecorder()
			e.ServeHTTP(listRec, listReq)
			if listRec.Code != http.StatusForbidden {
				t.Fatalf("expected 403 on student classrooms for %s, got %d", role, listRec.Code)
			}
		}
	})
}
