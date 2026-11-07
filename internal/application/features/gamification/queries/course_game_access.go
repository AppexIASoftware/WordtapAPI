package queries

import (
	"context"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// CourseGameAccess prevents standalone publication from bypassing a lesson entitlement.
type CourseGameAccess struct {
	lessons repositories.LessonRepository
	links   LinkedGameLessons
	policy  *access.ContentPolicy
}

type LinkedGameLessons interface {
	FindLessonsByGame(context.Context, string) ([]entities.LessonGame, error)
}

func NewCourseGameAccess(lessons repositories.LessonRepository, links LinkedGameLessons, policy *access.ContentPolicy) *CourseGameAccess {
	return &CourseGameAccess{lessons: lessons, links: links, policy: policy}
}

func (a *CourseGameAccess) authorize(ctx context.Context, game *entities.TeacherGame, studentID string) error {
	if a == nil || a.links == nil || game == nil || strings.TrimSpace(game.ID) == "" {
		return access.ErrContentDenied
	}
	links, err := a.links.FindLessonsByGame(ctx, game.ID)
	if err != nil {
		return err
	}
	for _, link := range links {
		if link.Lesson == nil || link.Lesson.ID != link.LessonID || link.TeacherGameID != game.ID {
			return access.ErrContentDenied
		}
		if err := a.policy.Authorize(ctx, link.Lesson.Course, link.Lesson, studentID, entities.RoleStudent); err != nil {
			return err
		}
	}
	if game.SourceType == "course_lesson" && game.SourceID != nil && strings.TrimSpace(*game.SourceID) != "" {
		if a.lessons == nil {
			return access.ErrContentDenied
		}
		lesson, err := a.lessons.FindByID(ctx, *game.SourceID)
		if err != nil {
			return err
		}
		if lesson == nil || lesson.ID != *game.SourceID {
			return access.ErrContentDenied
		}
		return a.policy.Authorize(ctx, lesson.Course, lesson, studentID, entities.RoleStudent)
	}
	return nil
}
