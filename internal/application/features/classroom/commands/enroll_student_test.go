package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestEnrollStudentHandler(t *testing.T) {
	repo := newMockClassroomRepo()
	handler := commands.NewEnrollStudentHandler(repo)

	activeClassroom := &entities.Classroom{
		ID:       "class-abc",
		Code:     "WT-B1-9420",
		Name:     "Aula de Conversación",
		IsActive: true,
	}
	repo.existingByCode["WT-B1-9420"] = activeClassroom

	t.Run("estudiante se inscribe exitosamente con código válido", func(t *testing.T) {
		cmd := commands.EnrollStudentCommand{
			Code:      "WT-B1-9420",
			StudentID: "student-1",
		}

		enrollment, err := handler.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}

		if enrollment == nil {
			t.Fatal("se esperaba una inscripción no nula")
		}

		if enrollment.ClassroomID != "class-abc" {
			t.Errorf("ClassroomID esperado 'class-abc', obtenido '%s'", enrollment.ClassroomID)
		}

		if enrollment.StudentID != "student-1" {
			t.Errorf("StudentID esperado 'student-1', obtenido '%s'", enrollment.StudentID)
		}
	})

	t.Run("falla con código inválido o inexistente", func(t *testing.T) {
		cmd := commands.EnrollStudentCommand{
			Code:      "CODIGO-INVENTADO",
			StudentID: "student-1",
		}

		_, err := handler.Handle(context.Background(), cmd)
		if err == nil {
			t.Fatal("se esperaba error al usar código no existente")
		}
	})
}
