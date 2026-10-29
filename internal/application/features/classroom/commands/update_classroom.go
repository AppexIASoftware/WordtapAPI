package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
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
		return nil, apperrors.New("El nombre del aula es obligatorio.")
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
		return nil, apperrors.New("No se encontró el aula.")
	}
	if classroom.TeacherID != cmd.TeacherID {
		return nil, apperrors.New("No tienes permisos para modificar esta aula.")
	}

	level := strings.TrimSpace(cmd.Level)
	if level != "" {
		classroom.Level = level
	}
	classroom.Name = name
	// Solo se sobrescribe cuando el campo viaja en la petición: un PUT parcial
	// (por ejemplo cambiar solo los minijuegos) no debe borrar la descripción.
	if cmd.Description != nil {
		classroom.Description = cmd.Description
	}

	if cmd.CourseIDs != nil {
		// The explicit list is authoritative, including an empty list.
		courses := make([]string, 0, len(cmd.CourseIDs))
		for _, id := range cmd.CourseIDs {
			if id = strings.TrimSpace(id); id != "" {
				courses = append(courses, id)
			}
		}
		assigned := strings.Join(courses, ",")
		classroom.AssignedCourses = &assigned
		classroom.CourseID = nil
		if len(courses) > 0 {
			classroom.CourseID = &courses[0]
		}
	} else if cmd.CourseID != nil {
		id := strings.TrimSpace(*cmd.CourseID)
		classroom.AssignedCourses = &id
		classroom.CourseID = nil
		if id != "" {
			classroom.CourseID = &id
		}
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
