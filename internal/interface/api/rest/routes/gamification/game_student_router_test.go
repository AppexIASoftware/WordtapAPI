package gamification_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	gameQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/gamification"
)

type mockStudentClassroomRepoForRouter struct {
	classrooms []entities.Classroom
}

func (m *mockStudentClassroomRepoForRouter) Create(ctx context.Context, c *entities.Classroom) error {
	return nil
}
func (m *mockStudentClassroomRepoForRouter) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepoForRouter) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepoForRouter) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepoForRouter) FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error) {
	var res []entities.Classroom
	for _, c := range m.classrooms {
		if c.IsActive {
			for _, e := range c.Enrollments {
				if e.StudentID == studentID {
					res = append(res, c)
					break
				}
			}
		}
	}
	return res, nil
}
func (m *mockStudentClassroomRepoForRouter) EnrollStudent(ctx context.Context, e *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockStudentClassroomRepoForRouter) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepoForRouter) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepoForRouter) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockStudentClassroomRepoForRouter) CreateStumble(ctx context.Context, s *entities.StudentStumble) error {
	return nil
}
func (m *mockStudentClassroomRepoForRouter) Update(ctx context.Context, c *entities.Classroom) error {
	return nil
}

func TestGameStudentRouter(t *testing.T) {
	course1 := "course-1"
	assignedJSON := `["course-1"]`

	classrooms := []entities.Classroom{
		{
			ID:        "cls-1",
			TeacherID: "teacher-1",
			CourseID:  &course1,
			IsActive:  true,
			Enrollments: []entities.ClassroomEnrollment{
				{StudentID: "student-100"},
			},
		},
	}

	games := []entities.TeacherGame{
		{
			ID:                    "game-general",
			TeacherID:             "teacher-1",
			Title:                 "Juego General",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "all_students",
		},
		{
			ID:                    "game-course-1",
			TeacherID:             "teacher-1",
			Title:                 "Juego Course 1",
			IsActive:              true,
			IsPublishedStandalone: true,
			AudienceScope:         "selected_courses",
			AssignedCourseIDs:     &assignedJSON,
		},
		{
			ID:                    "game-unpublished",
			TeacherID:             "teacher-1",
			Title:                 "Juego No Publicado",
			IsActive:              true,
			IsPublishedStandalone: false,
			AudienceScope:         "all_students",
		},
	}

	classroomRepo := &mockStudentClassroomRepoForRouter{classrooms: classrooms}
	gameRepo := newMockGameRouterRepo()
	gameRepo.games = games

	listHandler := gameQuery.NewListStudentTeacherGamesHandler(classroomRepo, gameRepo)
	getHandler := gameQuery.NewGetStudentTeacherGameDetailHandler(classroomRepo, gameRepo)
	studentRouter := gamification.NewGameStudentRouter(listHandler, getHandler)

	e := echo.New()
	v1 := e.Group("/api/v1")

	mockAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			uid := c.Request().Header.Get("X-Test-User-ID")
			if uid == "" {
				uid = "student-100"
			}
			c.Set("user_id", uid)
			c.Set("user_role", "student")
			return next(c)
		}
	}

	studentRouter.RegisterRoutes(v1, mockAuth)

	t.Run("GET /api/v1/games/student lista solo juegos publicados y autorizados", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/games/student", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200, obtenido %d", rec.Code)
		}

		var resp struct {
			Success bool             `json:"success"`
			Data    []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("error deserializando respuesta: %v", err)
		}

		if len(resp.Data) != 2 {
			t.Fatalf("se esperaban 2 juegos autorizados, se obtuvieron %d", len(resp.Data))
		}
	})

	t.Run("GET /api/v1/games/student/:id obtiene juego autorizado", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/games/student/game-general", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200, obtenido %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/games/student/:id deniega juego no publicado", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/games/student/game-unpublished", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("esperado 404, obtenido %d", rec.Code)
		}
	})
}
