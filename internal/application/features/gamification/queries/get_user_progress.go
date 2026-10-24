package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type GetUserProgressQuery struct {
	UserID string
}

type UserProgressSummaryDTO struct {
	CompletedLessonIDs []string `json:"completed_lesson_ids"`
	TotalStars         int      `json:"total_stars"`
	TotalPoints        int      `json:"total_points"`
	CurrentStreakDays  int      `json:"current_streak_days"`
	LongestStreakDays  int      `json:"longest_streak_days"`
	WordsLearned       int      `json:"words_learned"`
	LessonsCompleted   int      `json:"lessons_completed"`
	GamesCompleted     int      `json:"games_completed"`
	LivesRemaining     int      `json:"lives_remaining"`
	MaxLives           int      `json:"max_lives"`
	AccuracyPercent    float64  `json:"accuracy_percent"`
}

type GetUserProgressHandler struct {
	statsRepo repositories.UserLearningStatsRepository
}

func NewGetUserProgressHandler(statsRepo repositories.UserLearningStatsRepository) *GetUserProgressHandler {
	return &GetUserProgressHandler{statsRepo: statsRepo}
}

func (h *GetUserProgressHandler) Handle(ctx context.Context, query GetUserProgressQuery) (*UserProgressSummaryDTO, error) {
	if query.UserID == "" {
		return nil, errors.New("el ID de usuario es obligatorio")
	}

	stats, err := h.statsRepo.GetLearningStats(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	lessonIDs, err := h.statsRepo.GetCompletedLessonIDs(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	acc := stats.AccuracyPercent
	if acc <= 0 && stats.GamesCompleted > 0 {
		acc = 95.0
	}

	return &UserProgressSummaryDTO{
		CompletedLessonIDs: lessonIDs,
		TotalStars:         stats.TotalStars,
		TotalPoints:        stats.TotalPoints,
		CurrentStreakDays:  stats.CurrentStreakDays,
		LongestStreakDays:  stats.LongestStreakDays,
		WordsLearned:       stats.WordsLearned,
		LessonsCompleted:   stats.LessonsCompleted,
		GamesCompleted:     stats.GamesCompleted,
		LivesRemaining:     stats.LivesRemaining,
		MaxLives:           stats.MaxLives,
		AccuracyPercent:    acc,
	}, nil
}
