package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// UpdateClassroomCommand encapsula los parámetros para actualizar la configuración de un aula.
type UpdateClassroomCommand struct {
	ClassroomID     string   `json:"-"`
	TeacherID       string   `json:"-"`
	Name            string   `json:"name"`
	Level           string   `json:"level"`
	CourseID        *string  `json:"course_id"`
	CourseIDs       []string `json:"course_ids"`
	GameIDs         []string `json:"game_ids"`
	WeeklyGoalWords int      `json:"weekly_goal_words"`
	Description     *string  `json:"description"`
}

// UpdateClassroomHandler procesa la actualización de aulas virtuales con validación de pertenencia.
type UpdateClassroomHandler struct {
	classroomRepo repositories.ClassroomRepository
}

// NewUpdateClassroomHandler crea una instancia del handler para actualización de aulas.
func NewUpdateClassroomHandler(classroomRepo repositories.ClassroomRepository) *UpdateClassroomHandler {
	return &UpdateClassroomHandler{classroomRepo: classroomRepo}
}

// Handle valida los datos y aplica las modificaciones sobre la entidad Classroom en base de datos.
func (h *UpdateClassroomHandler) Handle(ctx context.Context, cmd UpdateClassroomCommand) (*entities.Classroom, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New("el nombre del aula es obligatorio")
	}
	if cmd.TeacherID == "" {
		return nil, errors.New("el identificador del docente es obligatorio")
	}
	if cmd.ClassroomID == "" {
		return nil, errors.New("el identificador del aula es obligatorio")
	}

	classroom, err := h.classroomRepo.FindByID(ctx, cmd.ClassroomID)
	if err != nil {
		return nil, err
	}
	if classroom == nil {
		return nil, errors.New("aula no encontrada")
	}
	if classroom.TeacherID != cmd.TeacherID {
		return nil, errors.New("no tiene permisos para modificar esta aula")
	}

	level := strings.TrimSpace(cmd.Level)
	if level != "" {
		classroom.Level = level
	}
	classroom.Name = name
	classroom.Description = cmd.Description

	if cmd.CourseID != nil && *cmd.CourseID != "" {
		classroom.CourseID = cmd.CourseID
	}

	if len(cmd.CourseIDs) > 0 {
		str := strings.Join(cmd.CourseIDs, ",")
		classroom.AssignedCourses = &str
	}

	if len(cmd.GameIDs) > 0 {
		str := strings.Join(cmd.GameIDs, ",")
		classroom.ActiveGames = &str
	}

	if cmd.WeeklyGoalWords > 0 {
		classroom.WeeklyGoalWords = cmd.WeeklyGoalWords
	}
	classroom.UpdatedAt = time.Now()

	if err := h.classroomRepo.Update(ctx, classroom); err != nil {
		return nil, err
	}

	return classroom, nil
}
