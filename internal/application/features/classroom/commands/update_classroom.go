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

// UpdateClassroomCommand encapsulates parameters to update classroom configuration.
type UpdateClassroomCommand struct {
	ClassroomID     string   `json:"-"`
	TeacherID       string   `json:"-"`
	Name            string   `json:"name"`
	Level           string   `json:"level"`
	CourseID        *string  `json:"course_id"`
	CourseIDs       []string `json:"course_ids"`
	LessonIDs       []string `json:"lesson_ids"`
	GameIDs         []string `json:"game_ids"`
	WeeklyGoalWords int      `json:"weekly_goal_words"`
	Description     *string  `json:"description"`
}

// UpdateClassroomHandler processes classroom updates with ownership validation.
type UpdateClassroomHandler struct {
	classroomRepo repositories.ClassroomRepository
	courseRepo    CourseLookup
}

// NewUpdateClassroomHandler creates an instance of the handler for classroom updates.
func NewUpdateClassroomHandler(classroomRepo repositories.ClassroomRepository, courses ...CourseLookup) *UpdateClassroomHandler {
	var lookup CourseLookup
	if len(courses) > 0 {
		lookup = courses[0]
	}
	return &UpdateClassroomHandler{classroomRepo: classroomRepo, courseRepo: lookup}
}

// Handle validates data and applies modifications to the Classroom entity in the database.
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
	ids := cmd.CourseIDs
	if ids == nil && cmd.CourseID != nil {
		ids = []string{*cmd.CourseID}
	}
	if err := validateCourses(ctx, h.courseRepo, cmd.TeacherID, ids); err != nil {
		return nil, err
	}
	if cmd.LessonIDs != nil {
		if ids == nil {
			if classroom.AssignedCourses != nil {
				ids = strings.Split(*classroom.AssignedCourses, ",")
			} else if classroom.CourseID != nil {
				ids = []string{*classroom.CourseID}
			}
			if err := validateCourses(ctx, h.courseRepo, cmd.TeacherID, ids); err != nil {
				return nil, err
			}
		}
		selection, err := encodeLessonSelection(ctx, h.courseRepo, ids, cmd.LessonIDs)
		if err != nil {
			return nil, err
		}
		classroom.AssignedLessons = selection
	} else if ids != nil && classroom.AssignedLessons != nil {
		// An old client changing courses cannot leave lessons from removed courses behind.
		selection, err := retainAssignedLessons(ctx, h.courseRepo, ids, classroom.AssignedLessons)
		if err != nil {
			return nil, err
		}
		classroom.AssignedLessons = selection
	}

	level := strings.TrimSpace(cmd.Level)
	if level != "" {
		classroom.Level = level
	}
	classroom.Name = name
	// Only overwrite this field when it is present in the request: a partial PUT
	// (for example, changing only the minigames) must not clear the description.
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
