package queries

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListStudentTeacherGamesQuery struct {
	StudentID string
}

type ListStudentTeacherGamesHandler struct {
	classroomRepo repositories.ClassroomRepository
	gameRepo      repositories.TeacherGameRepository
}

func NewListStudentTeacherGamesHandler(
	classroomRepo repositories.ClassroomRepository,
	gameRepo repositories.TeacherGameRepository,
) *ListStudentTeacherGamesHandler {
	return &ListStudentTeacherGamesHandler{
		classroomRepo: classroomRepo,
		gameRepo:      gameRepo,
	}
}

func (h *ListStudentTeacherGamesHandler) Handle(ctx context.Context, q ListStudentTeacherGamesQuery) ([]entities.TeacherGame, error) {
	studentID := strings.TrimSpace(q.StudentID)
	if studentID == "" {
		return []entities.TeacherGame{}, nil
	}

	classrooms, err := h.classroomRepo.FindByStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if len(classrooms) == 0 {
		return []entities.TeacherGame{}, nil
	}

	teacherIDs := make([]string, 0)
	teacherSeen := make(map[string]bool)
	enrolledCoursesByTeacher := make(map[string]map[string]bool)

	for _, c := range classrooms {
		tid := strings.TrimSpace(c.TeacherID)
		if tid == "" {
			continue
		}
		if !teacherSeen[tid] {
			teacherSeen[tid] = true
			teacherIDs = append(teacherIDs, tid)
			enrolledCoursesByTeacher[tid] = make(map[string]bool)
		}

		if c.CourseID != nil && strings.TrimSpace(*c.CourseID) != "" {
			enrolledCoursesByTeacher[tid][strings.TrimSpace(*c.CourseID)] = true
		}

		if c.AssignedCourses != nil {
			raw := strings.TrimSpace(*c.AssignedCourses)
			if raw != "" {
				var ids []string
				if err := json.Unmarshal([]byte(raw), &ids); err == nil {
					for _, id := range ids {
						if clean := strings.TrimSpace(id); clean != "" {
							enrolledCoursesByTeacher[tid][clean] = true
						}
					}
				} else {
					for _, part := range strings.Split(raw, ",") {
						if clean := strings.TrimSpace(part); clean != "" {
							enrolledCoursesByTeacher[tid][clean] = true
						}
					}
				}
			}
		}
	}

	if len(teacherIDs) == 0 {
		return []entities.TeacherGame{}, nil
	}

	games, err := h.gameRepo.FindStandaloneByTeachers(ctx, teacherIDs)
	if err != nil {
		return nil, err
	}

	result := make([]entities.TeacherGame, 0, len(games))
	for _, g := range games {
		if !g.IsActive || !g.IsPublishedStandalone {
			continue
		}

		enrolledCourses, ok := enrolledCoursesByTeacher[g.TeacherID]
		if !ok {
			continue
		}

		scope := strings.TrimSpace(g.AudienceScope)
		if scope == "" || scope == "all_students" {
			result = append(result, g)
			continue
		}

		if scope == "selected_courses" {
			if g.AssignedCourseIDs == nil {
				continue
			}
			raw := strings.TrimSpace(*g.AssignedCourseIDs)
			if raw == "" {
				continue
			}

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

			if hasAccess {
				result = append(result, g)
			}
		}
	}

	return result, nil
}
