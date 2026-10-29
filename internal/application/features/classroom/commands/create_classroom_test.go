package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockClassroomRepo struct {
	createdClassroom   *entities.Classroom
	createdEnrollment  *entities.ClassroomEnrollment
	createdStumbles    []entities.StudentStumble
	existingByCode     map[string]*entities.Classroom
	existingEnrollment *entities.ClassroomEnrollment
	classrooms         []entities.Classroom
}

func newMockClassroomRepo() *mockClassroomRepo {
	return &mockClassroomRepo{
		existingByCode:  make(map[string]*entities.Classroom),
		createdStumbles: make([]entities.StudentStumble, 0),
	}
}

func (m *mockClassroomRepo) Create(ctx context.Context, classroom *entities.Classroom) error {
	m.createdClassroom = classroom
	return nil
}

func (m *mockClassroomRepo) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	return nil, nil
}

func (m *mockClassroomRepo) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	return m.existingByCode[code], nil
}

func (m *mockClassroomRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	return m.classrooms, nil
}

func (m *mockClassroomRepo) FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error) {
	return m.classrooms, nil
}

func (m *mockClassroomRepo) EnrollStudent(ctx context.Context, enrollment *entities.ClassroomEnrollment) error {
	m.createdEnrollment = enrollment
	return nil
}

func (m *mockClassroomRepo) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	return m.existingEnrollment, nil
}

func (m *mockClassroomRepo) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}

func (m *mockClassroomRepo) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	return m.createdStumbles, nil
}

func (m *mockClassroomRepo) CreateStumble(ctx context.Context, stumble *entities.StudentStumble) error {
	m.createdStumbles = append(m.createdStumbles, *stumble)
	return nil
}

func (m *mockClassroomRepo) Update(ctx context.Context, classroom *entities.Classroom) error {
	return nil
}

func TestCreateClassroomHandler(t *testing.T) {
	repo := newMockClassroomRepo()
	handler := commands.NewCreateClassroomHandler(repo)

	t.Run("exitoso con generación de código WT-B1", func(t *testing.T) {
		cmd := commands.CreateClassroomCommand{
			TeacherID: "teacher-123",
			Name:      "Inglés B1 - Cohorte Matutina",
			Level:     "B1 Intermediate",
		}

		classroom, err := handler.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}

		if classroom == nil {
			t.Fatal("se esperaba que el aula no fuera nula")
		}

		if classroom.Name != "Inglés B1 - Cohorte Matutina" {
			t.Errorf("nombre esperado 'Inglés B1 - Cohorte Matutina', obtenido '%s'", classroom.Name)
		}

		if len(classroom.Code) < 8 || classroom.Code[:5] != "WT-B1" {
			t.Errorf("formato de código inválido, obtenido '%s'", classroom.Code)
		}

		if len(repo.createdStumbles) == 0 {
			t.Error("se esperaba que sembrara items pedagógicos de tropiezo por defecto")
		}
	})

	t.Run("falla cuando el nombre está vacío", func(t *testing.T) {
		cmd := commands.CreateClassroomCommand{
			TeacherID: "teacher-123",
			Name:      "   ",
		}

		_, err := handler.Handle(context.Background(), cmd)
		if err == nil {
			t.Fatal("se esperaba error con nombre vacío")
		}
	})

	t.Run("falla cuando teacherID está vacío", func(t *testing.T) {
		cmd := commands.CreateClassroomCommand{
			TeacherID: "",
			Name:      "Aula Sin Docente",
		}

		_, err := handler.Handle(context.Background(), cmd)
		if err == nil {
			t.Fatal("se esperaba error con docente vacío")
		}
	})
}
