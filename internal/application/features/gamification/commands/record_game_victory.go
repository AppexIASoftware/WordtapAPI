package commands

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type RecordGameVictoryCommand struct {
	UserID         string
	LessonID       string
	StarsEarned    int
	WordsPracticed int
}

type RecordGameVictoryHandler struct {
	statsRepo repositories.UserLearningStatsRepository
}

func NewRecordGameVictoryHandler(statsRepo repositories.UserLearningStatsRepository) *RecordGameVictoryHandler {
	return &RecordGameVictoryHandler{statsRepo: statsRepo}
}

func (h *RecordGameVictoryHandler) Handle(ctx context.Context, cmd RecordGameVictoryCommand) (*repositories.RecordProgressResult, error) {
	if cmd.UserID == "" {
		return nil, errors.New("el ID de usuario es obligatorio")
	}

	stars := cmd.StarsEarned
	if stars < 0 {
		stars = 0
	} else if stars > 3 {
		stars = 3
	}

	words := cmd.WordsPracticed
	if words < 0 {
		words = 0
	}

	return h.statsRepo.RecordGameVictory(ctx, repositories.RecordProgressParams{
		UserID:         cmd.UserID,
		LessonID:       cmd.LessonID,
		StarsEarned:    stars,
		WordsPracticed: words,
	})
}
