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

type mockClassroomRepo struct {
	classrooms []entities.Classroom
	byCode     map[string]*entities.Classroom
	stumbles   []entities.StudentStumble
}

func newMockClassroomRepo() *mockClassroomRepo {
	return &mockClassroomRepo{
		classrooms: make([]entities.Classroom, 0),
		byCode:     make(map[string]*entities.Classroom),
		stumbles:   make([]entities.StudentStumble, 0),
	}
}

func (m *mockClassroomRepo) Create(ctx context.Context, c *entities.Classroom) error {
	m.classrooms = append(m.classrooms, *c)
	m.byCode[c.Code] = c
	return nil
}

func (m *mockClassroomRepo) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	for _, c := range m.classrooms {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockClassroomRepo) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	return m.byCode[code], nil
}

func (m *mockClassroomRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	var list []entities.Classroom
	for _, c := range m.classrooms {
		if c.TeacherID == teacherID {
			list = append(list, c)
		}
	}
	return list, nil
}

func (m *mockClassroomRepo) FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error) {
	return nil, nil
}

func (m *mockClassroomRepo) EnrollStudent(ctx context.Context, e *entities.ClassroomEnrollment) error {
	return nil
}

func (m *mockClassroomRepo) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}

func (m *mockClassroomRepo) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}

func (m *mockClassroomRepo) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	return m.stumbles, nil
}

func (m *mockClassroomRepo) CreateStumble(ctx context.Context, s *entities.StudentStumble) error {
	m.stumbles = append(m.stumbles, *s)
	return nil
}

func (m *mockClassroomRepo) Update(ctx context.Context, c *entities.Classroom) error {
	for i, existing := range m.classrooms {
		if existing.ID == c.ID {
			m.classrooms[i] = *c
			return nil
		}
	}
	return nil
}

func setupClassroomRouter() (*echo.Echo, *mockClassroomRepo) {
	e := echo.New()
	repo := newMockClassroomRepo()

	createHandler := classroomCmd.NewCreateClassroomHandler(repo)
	updateHandler := classroomCmd.NewUpdateClassroomHandler(repo)
	enrollHandler := classroomCmd.NewEnrollStudentHandler(repo)
	listHandler := classroomQuery.NewListTeacherClassroomsHandler(repo)
	stumblesHandler := classroomQuery.NewGetStumbleListHandler(repo)

	r := classroom.NewClassroomRouter(createHandler, updateHandler, enrollHandler, listHandler, stumblesHandler)

	v1 := e.Group("/api/v1")
	passMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-uuid-1")
			return next(c)
		}
	}
	r.RegisterRoutes(v1, passMiddleware, passMiddleware)

	return e, repo
}

func TestClassroomRouter_CreateAndList(t *testing.T) {
	e, repo := setupClassroomRouter()

	t.Run("POST /api/v1/teacher/classrooms crea aula exitosamente", func(t *testing.T) {
		payload := map[string]any{
			"name":  "Inglés B1 Intensivo",
			"level": "B1 Intermediate",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/classrooms", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("código de estado esperado %d, obtenido %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /api/v1/teacher/classrooms lista aulas", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/classrooms", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código de estado esperado %d, obtenido %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("PUT /api/v1/teacher/classrooms/:id actualiza aula", func(t *testing.T) {
		// Crear aula previa en el repo mock
		_ = repo.Create(context.Background(), &entities.Classroom{
			ID:        "cls-100",
			TeacherID: "teacher-uuid-1",
			Name:      "Aula Original",
			Level:     "A2",
			Code:      "WT-A2-9999",
		})

		updatePayload := map[string]any{
			"name":              "Aula Actualizada",
			"level":             "B1 Intermediate",
			"weekly_goal_words": 60,
		}
		body, _ := json.Marshal(updatePayload)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/teacher/classrooms/cls-100", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código de estado esperado %d, obtenido %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}
