package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type EnrollStudentCommand struct {
	Code      string `json:"code"`
	StudentID string `json:"-"`
}

type EnrollStudentHandler struct {
	classroomRepo repositories.ClassroomRepository
}

func NewEnrollStudentHandler(classroomRepo repositories.ClassroomRepository) *EnrollStudentHandler {
	return &EnrollStudentHandler{classroomRepo: classroomRepo}
}

func (h *EnrollStudentHandler) Handle(ctx context.Context, cmd EnrollStudentCommand) (*entities.ClassroomEnrollment, error) {
	code := strings.TrimSpace(cmd.Code)
	if code == "" {
		return nil, errors.New("el código del aula es obligatorio")
	}
	if cmd.StudentID == "" {
		return nil, errors.New("el identificador del estudiante es obligatorio")
	}

	classroom, err := h.classroomRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if classroom == nil || !classroom.IsActive {
		return nil, errors.New("el código ingresado no corresponde a un aula activa")
	}

	existing, err := h.classroomRepo.FindEnrollment(ctx, classroom.ID, cmd.StudentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	enrollment := &entities.ClassroomEnrollment{
		ClassroomID:   classroom.ID,
		StudentID:     cmd.StudentID,
		EnrolledAt:    now,
		LastActiveAt:  &now,
		WordsMastered: 0,
		AccuracyPct:   100,
		StreakDays:    1,
	}

	if err := h.classroomRepo.EnrollStudent(ctx, enrollment); err != nil {
		return nil, err
	}

	return enrollment, nil
}
