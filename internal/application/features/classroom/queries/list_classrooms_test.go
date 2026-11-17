package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockClassroomQueryRepo struct {
	classrooms []entities.Classroom
	stumbles   []entities.StudentStumble
}

func (m *mockClassroomQueryRepo) Create(ctx context.Context, classroom *entities.Classroom) error {
	return nil
}

func (m *mockClassroomQueryRepo) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	for _, c := range m.classrooms {
		if c.ID == id {
			cls := c
			return &cls, nil
		}
	}
	return nil, nil
}

func (m *mockClassroomQueryRepo) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	return nil, nil
}

func (m *mockClassroomQueryRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	return m.classrooms, nil
}

func (m *mockClassroomQueryRepo) FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error) {
	return m.classrooms, nil
}

func (m *mockClassroomQueryRepo) EnrollStudent(ctx context.Context, enrollment *entities.ClassroomEnrollment) error {
	return nil
}

func (m *mockClassroomQueryRepo) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}

func (m *mockClassroomQueryRepo) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}

func (m *mockClassroomQueryRepo) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	return m.stumbles, nil
}

func (m *mockClassroomQueryRepo) CreateStumble(ctx context.Context, stumble *entities.StudentStumble) error {
	return nil
}

func (m *mockClassroomQueryRepo) Update(ctx context.Context, classroom *entities.Classroom) error {
	return nil
}

func TestListTeacherClassroomsHandler(t *testing.T) {
	repo := &mockClassroomQueryRepo{
		classrooms: []entities.Classroom{
			{ID: "c-1", Name: "Aula B1", TeacherID: "teacher-1"},
			{ID: "c-2", Name: "Aula B2", TeacherID: "teacher-1"},
		},
	}
	handler := queries.NewListTeacherClassroomsHandler(repo)

	t.Run("lista aulas del docente", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListTeacherClassroomsQuery{TeacherID: "teacher-1"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("se esperaban 2 aulas, obtenidas %d", len(res))
		}
	})

	t.Run("error si teacherID es vacío", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), queries.ListTeacherClassroomsQuery{TeacherID: ""})
		if err == nil {
			t.Fatal("se esperaba error con teacherID vacío")
		}
	})
}

func TestGetStumbleListHandler(t *testing.T) {
	repo := &mockClassroomQueryRepo{
		classrooms: []entities.Classroom{
			{ID: "c-1", TeacherID: "teacher-1", Name: "Aula B1"},
		},
		stumbles: []entities.StudentStumble{
			{ID: "s-1", Term: "thought", FailRate: 68},
			{ID: "s-2", Term: "easy word", FailRate: 15}, // menos del 40%, no debe incluirse
			{ID: "s-3", Term: "through", FailRate: 55},
		},
	}
	handler := queries.NewGetStumbleListHandler(repo)

	t.Run("filtra solo términos con tasa de fallo >= 40%", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.GetStumbleListQuery{
			ClassroomID:   "c-1",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("se esperaban 2 items críticos con tasa >= 40%%, obtenidos %d", len(res))
		}
	})

	t.Run("rechaza docente ajeno con ErrClassroomForbidden", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), queries.GetStumbleListQuery{
			ClassroomID:   "c-1",
			RequesterID:   "teacher-2",
			RequesterRole: entities.RoleInstructor,
		})
		if !errors.Is(err, queries.ErrClassroomForbidden) {
			t.Fatalf("expected ErrClassroomForbidden, got %v", err)
		}
	})

	t.Run("devuelve ErrClassroomNotFound para aula inexistente", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), queries.GetStumbleListQuery{
			ClassroomID:   "c-missing",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if !errors.Is(err, queries.ErrClassroomNotFound) {
			t.Fatalf("expected ErrClassroomNotFound, got %v", err)
		}
	})
}
