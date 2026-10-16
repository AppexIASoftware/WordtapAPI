package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockUpdateClassroomRepo struct {
	classroom *entities.Classroom
	updated   bool
}

func (m *mockUpdateClassroomRepo) Create(ctx context.Context, c *entities.Classroom) error {
	return nil
}

func (m *mockUpdateClassroomRepo) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	if m.classroom != nil && m.classroom.ID == id {
		return m.classroom, nil
	}
	return nil, nil
}

func (m *mockUpdateClassroomRepo) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	return nil, nil
}

func (m *mockUpdateClassroomRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	return nil, nil
}

func (m *mockUpdateClassroomRepo) EnrollStudent(ctx context.Context, e *entities.ClassroomEnrollment) error {
	return nil
}

func (m *mockUpdateClassroomRepo) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}

func (m *mockUpdateClassroomRepo) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}

func (m *mockUpdateClassroomRepo) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	return nil, nil
}

func (m *mockUpdateClassroomRepo) CreateStumble(ctx context.Context, s *entities.StudentStumble) error {
	return nil
}

func (m *mockUpdateClassroomRepo) Update(ctx context.Context, c *entities.Classroom) error {
	m.classroom = c
	m.updated = true
	return nil
}

func TestUpdateClassroomHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("Actualiza aula exitosamente", func(t *testing.T) {
		repo := &mockUpdateClassroomRepo{
			classroom: &entities.Classroom{
				ID:        "cls-1",
				TeacherID: "teacher-1",
				Name:      "Aula Original",
				Level:     "A2",
			},
		}

		handler := commands.NewUpdateClassroomHandler(repo)
		desc := "Nueva descripción"
		cmd := commands.UpdateClassroomCommand{
			ClassroomID:     "cls-1",
			TeacherID:       "teacher-1",
			Name:            "Aula Actualizada",
			Level:           "B1 Intermediate",
			Description:     &desc,
			CourseIDs:       []string{"course-1", "course-2"},
			GameIDs:         []string{"match", "quiz"},
			WeeklyGoalWords: 80,
		}

		result, err := handler.Handle(ctx, cmd)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if result.Name != "Aula Actualizada" {
			t.Errorf("nombre esperado 'Aula Actualizada', obtenido %s", result.Name)
		}
		if result.Level != "B1 Intermediate" {
			t.Errorf("nivel esperado 'B1 Intermediate', obtenido %s", result.Level)
		}
		if !repo.updated {
			t.Errorf("esperaba que el repositorio registrara la actualización")
		}
	})

	t.Run("Preserva la descripción cuando la petición no la envía", func(t *testing.T) {
		original := "Descripción original del aula"
		repo := &mockUpdateClassroomRepo{
			classroom: &entities.Classroom{
				ID:          "cls-1",
				TeacherID:   "teacher-1",
				Name:        "Aula Original",
				Description: &original,
			},
		}

		handler := commands.NewUpdateClassroomHandler(repo)
		cmd := commands.UpdateClassroomCommand{
			ClassroomID: "cls-1",
			TeacherID:   "teacher-1",
			Name:        "Aula Original",
			GameIDs:     []string{"match", "custom_game-1"},
		}

		result, err := handler.Handle(ctx, cmd)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if result.Description == nil || *result.Description != original {
			t.Errorf("la descripción debía conservarse, obtenido %v", result.Description)
		}
		if result.ActiveGames == nil || *result.ActiveGames != "match,custom_game-1" {
			t.Errorf("minijuegos esperados 'match,custom_game-1', obtenido %v", result.ActiveGames)
		}
	})

	t.Run("Falla si docente no es propietario", func(t *testing.T) {
		repo := &mockUpdateClassroomRepo{
			classroom: &entities.Classroom{
				ID:        "cls-1",
				TeacherID: "teacher-other",
				Name:      "Aula",
			},
		}

		handler := commands.NewUpdateClassroomHandler(repo)
		cmd := commands.UpdateClassroomCommand{
			ClassroomID: "cls-1",
			TeacherID:   "teacher-1",
			Name:        "Nombre",
		}

		_, err := handler.Handle(ctx, cmd)
		if err == nil {
			t.Fatalf("se esperaba error de autorización")
		}
	})

	t.Run("Falla si nombre está vacío", func(t *testing.T) {
		repo := &mockUpdateClassroomRepo{}
		handler := commands.NewUpdateClassroomHandler(repo)
		cmd := commands.UpdateClassroomCommand{
			ClassroomID: "cls-1",
			TeacherID:   "teacher-1",
			Name:        "   ",
		}

		_, err := handler.Handle(ctx, cmd)
		if err == nil {
			t.Fatalf("se esperaba error de validación de nombre")
		}
	})
}
