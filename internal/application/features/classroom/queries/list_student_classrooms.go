package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListStudentClassroomsQuery struct {
	StudentID string
}

type ListStudentClassroomsHandler struct {
	classroomRepo repositories.ClassroomRepository
}

func NewListStudentClassroomsHandler(classroomRepo repositories.ClassroomRepository) *ListStudentClassroomsHandler {
	return &ListStudentClassroomsHandler{classroomRepo: classroomRepo}
}

func (h *ListStudentClassroomsHandler) Handle(ctx context.Context, query ListStudentClassroomsQuery) ([]entities.Classroom, error) {
	if query.StudentID == "" {
		return nil, errors.New("el identificador del estudiante es obligatorio")
	}

	return h.classroomRepo.FindByStudent(ctx, query.StudentID)
}
