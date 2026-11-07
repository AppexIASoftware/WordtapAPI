package queries_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestListStudentClassroomsHandler(t *testing.T) {
	repo := &mockClassroomQueryRepo{
		classrooms: []entities.Classroom{
			{ID: "c-10", Name: "Inglés B1 — Mañana", Code: "WT-4B7K"},
			{ID: "c-20", Name: "Conversación C1", Code: "WT-9Q2M"},
		},
	}
	handler := queries.NewListStudentClassroomsHandler(repo)

	t.Run("lista aulas donde el estudiante está matriculado", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), queries.ListStudentClassroomsQuery{StudentID: "student-123"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("se esperaban 2 aulas, obtenidas %d", len(res))
		}
		if res[0].Code != "WT-4B7K" {
			t.Errorf("código inesperado: %s", res[0].Code)
		}
	})

	t.Run("rechaza studentID vacío", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), queries.ListStudentClassroomsQuery{StudentID: ""})
		if err == nil {
			t.Fatal("se esperaba error con studentID vacío")
		}
	})
}
