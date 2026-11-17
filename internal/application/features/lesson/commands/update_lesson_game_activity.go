package commands

import (
	"context"
	"strings"
	"time"

	lessonaccess "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type UpdateLessonGameActivityCommand struct {
	LessonID         string
	GameID           string
	RequesterID      string
	RequesterRole    entities.UserRole
	BaseArchetype    string
	Title            string
	Description      *string
	TimeLimitSeconds int
	AllowedMistakes  int
	RewardXP         int
	ItemCount        int
}

type UpdateLessonGameActivityHandler struct {
	lessons  repositories.LessonRepository
	courses  repositories.CourseRepository
	gameRepo repositories.TeacherGameRepository
	links    repositories.LessonGameRepository
}

func NewUpdateLessonGameActivityHandler(
	lessons repositories.LessonRepository,
	courses repositories.CourseRepository,
	gameRepo repositories.TeacherGameRepository,
) *UpdateLessonGameActivityHandler {
	return &UpdateLessonGameActivityHandler{
		lessons:  lessons,
		courses:  courses,
		gameRepo: gameRepo,
	}
}

// WithLinkRepo attaches LessonGameRepository to verify the game is linked to the lesson before mutating.
func (h *UpdateLessonGameActivityHandler) WithLinkRepo(links repositories.LessonGameRepository) *UpdateLessonGameActivityHandler {
	h.links = links
	return h
}

func (h *UpdateLessonGameActivityHandler) Handle(
	ctx context.Context,
	cmd UpdateLessonGameActivityCommand,
) (*entities.TeacherGame, error) {
	_, course, err := loadLessonCourse(ctx, h.lessons, h.courses, cmd.LessonID)
	if err != nil {
		return nil, err
	}
	if !lessonaccess.CanLinkGameToCourse(course, cmd.RequesterID, cmd.RequesterRole) {
		return nil, ErrLessonGameAccessDenied
	}
	if course.Status == entities.ContentStatusInReview {
		return nil, ErrLessonGameCourseInReview
	}

	if h.links != nil {
		link, err := h.links.FindByLessonAndGame(ctx, cmd.LessonID, cmd.GameID)
		if err != nil {
			return nil, err
		}
		if link == nil {
			return nil, ErrLessonGameLinkNotFound
		}
	}

	game, err := h.gameRepo.FindByID(ctx, cmd.GameID)
	if err != nil || game == nil {
		return nil, ErrLessonGameGameNotFound
	}
	if game.IsGlobal || game.OwnerType == entities.OwnerTypePlatform {
		if cmd.RequesterRole != entities.RoleAdmin {
			return nil, ErrLessonGameAccessDenied
		}
	} else if cmd.RequesterRole != entities.RoleInstructor || game.TeacherID != cmd.RequesterID {
		return nil, ErrLessonGameAccessDenied
	}
	if game.SourceType == "course_lesson" && game.SourceID != nil && strings.TrimSpace(*game.SourceID) != "" && *game.SourceID != cmd.LessonID {
		return nil, ErrLessonGameAccessDenied
	}

	archetype := strings.ToLower(strings.TrimSpace(cmd.BaseArchetype))
	if archetype != "" {
		game.BaseArchetype = archetype
	}

	title := strings.TrimSpace(cmd.Title)
	if title != "" {
		game.Title = title
	}

	if cmd.Description != nil {
		trimmedDesc := strings.TrimSpace(*cmd.Description)
		if trimmedDesc == "" {
			game.Description = nil
		} else {
			game.Description = &trimmedDesc
		}
	}

	if cmd.TimeLimitSeconds > 0 {
		game.TimeLimitSeconds = cmd.TimeLimitSeconds
	}
	if cmd.AllowedMistakes >= 0 {
		game.AllowedMistakes = cmd.AllowedMistakes
	}
	if cmd.RewardXP > 0 {
		game.RewardXP = cmd.RewardXP
	}
	if cmd.ItemCount > 0 {
		game.ItemCount = cmd.ItemCount
	}

	game.UpdatedAt = time.Now()

	if err := h.gameRepo.Update(ctx, game); err != nil {
		return nil, err
	}

	return game, nil
}
