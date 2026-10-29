package queries

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type GetStudentTeacherGameDetailQuery struct {
	StudentID string
	GameID    string
}

type GetStudentTeacherGameDetailHandler struct {
	classroomRepo repositories.ClassroomRepository
	gameRepo      repositories.TeacherGameRepository
}

func NewGetStudentTeacherGameDetailHandler(
	classroomRepo repositories.ClassroomRepository,
	gameRepo repositories.TeacherGameRepository,
) *GetStudentTeacherGameDetailHandler {
	return &GetStudentTeacherGameDetailHandler{
		classroomRepo: classroomRepo,
		gameRepo:      gameRepo,
	}
}

func (h *GetStudentTeacherGameDetailHandler) Handle(ctx context.Context, q GetStudentTeacherGameDetailQuery) (*entities.TeacherGame, error) {
	studentID := strings.TrimSpace(q.StudentID)
	if studentID == "" {
		return nil, errors.New("el identificador del estudiante es obligatorio")
	}
	gameID := strings.TrimSpace(q.GameID)
	if gameID == "" {
		return nil, errors.New("el identificador del juego es obligatorio")
	}

	game, err := h.gameRepo.FindByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game == nil || !game.IsActive || !game.IsPublishedStandalone {
		return nil, apperrors.New("No se encontró la actividad o no está disponible.")
	}

	classrooms, err := h.classroomRepo.FindByStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}

	isEnrolledWithTeacher := false
	enrolledCourses := make(map[string]bool)

	for _, c := range classrooms {
		if strings.TrimSpace(c.TeacherID) == game.TeacherID {
			isEnrolledWithTeacher = true
			if c.CourseID != nil && strings.TrimSpace(*c.CourseID) != "" {
				enrolledCourses[strings.TrimSpace(*c.CourseID)] = true
			}
			if c.AssignedCourses != nil {
				raw := strings.TrimSpace(*c.AssignedCourses)
				if raw != "" {
					var ids []string
					if err := json.Unmarshal([]byte(raw), &ids); err == nil {
						for _, id := range ids {
							if clean := strings.TrimSpace(id); clean != "" {
								enrolledCourses[clean] = true
							}
						}
					} else {
						for _, part := range strings.Split(raw, ",") {
							if clean := strings.TrimSpace(part); clean != "" {
								enrolledCourses[clean] = true
							}
						}
					}
				}
			}
		}
	}

	if !isEnrolledWithTeacher {
		return nil, apperrors.New("No tienes permisos para acceder a esta actividad.")
	}

	scope := strings.TrimSpace(game.AudienceScope)
	if scope == "selected_courses" {
		if game.AssignedCourseIDs == nil {
			return nil, apperrors.New("No tienes permisos para acceder a esta actividad.")
		}
		raw := strings.TrimSpace(*game.AssignedCourseIDs)
		var targetCourseIDs []string
		if err := json.Unmarshal([]byte(raw), &targetCourseIDs); err != nil {
			for _, part := range strings.Split(raw, ",") {
				if clean := strings.TrimSpace(part); clean != "" {
					targetCourseIDs = append(targetCourseIDs, clean)
				}
			}
		}

		hasAccess := false
		for _, targetID := range targetCourseIDs {
			if enrolledCourses[strings.TrimSpace(targetID)] {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, apperrors.New("No tienes permisos para acceder a esta actividad.")
		}
	}

	return game, nil
}
