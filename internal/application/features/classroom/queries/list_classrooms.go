package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListTeacherClassroomsQuery struct {
	TeacherID string
}

type ListTeacherClassroomsHandler struct {
	classroomRepo repositories.ClassroomRepository
}

func NewListTeacherClassroomsHandler(classroomRepo repositories.ClassroomRepository) *ListTeacherClassroomsHandler {
	return &ListTeacherClassroomsHandler{classroomRepo: classroomRepo}
}

func (h *ListTeacherClassroomsHandler) Handle(ctx context.Context, query ListTeacherClassroomsQuery) ([]entities.Classroom, error) {
	if query.TeacherID == "" {
		return nil, errors.New("el identificador del docente es obligatorio")
	}

	return h.classroomRepo.FindByTeacher(ctx, query.TeacherID)
}
