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

	game, err := h.gameRepo.FindByID(ctx, cmd.GameID)
	if err != nil || game == nil {
		return nil, ErrLessonGameGameNotFound
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
