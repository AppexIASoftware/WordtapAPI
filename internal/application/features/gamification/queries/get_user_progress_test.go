package queries_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type mockProgressStatsRepo struct {
	stats              *entities.UserLearningStats
	completedLessonIDs []string
}

func (m *mockProgressStatsRepo) GetLearningStats(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	if m.stats == nil {
		m.stats = &entities.UserLearningStats{
			UserID:            userID,
			TotalStars:        12,
			TotalPoints:       120,
			CurrentStreakDays: 3,
			WordsLearned:      18,
			LessonsCompleted:  2,
			GamesCompleted:    4,
			LivesRemaining:    4,
			MaxLives:          5,
		}
	}
	return m.stats, nil
}

func (m *mockProgressStatsRepo) DeductUserLife(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	return nil, nil
}

func (m *mockProgressStatsRepo) RefillUserLives(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	return nil, nil
}

func (m *mockProgressStatsRepo) RecordGameVictory(ctx context.Context, params repositories.RecordProgressParams) (*repositories.RecordProgressResult, error) {
	return nil, nil
}

func (m *mockProgressStatsRepo) GetCompletedLessonIDs(ctx context.Context, userID string) ([]string, error) {
	return m.completedLessonIDs, nil
}

func TestGetUserProgressHandler(t *testing.T) {
	repo := &mockProgressStatsRepo{
		completedLessonIDs: []string{"lesson-1", "lesson-2"},
	}
	handler := queries.NewGetUserProgressHandler(repo)

	t.Run("fails when user_id is empty", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), queries.GetUserProgressQuery{
			UserID: "",
		})
		if err == nil {
			t.Fatalf("expected error for empty user_id")
		}
	})

	t.Run("returns full progress summary", func(t *testing.T) {
		summary, err := handler.Handle(context.Background(), queries.GetUserProgressQuery{
			UserID: "user-123",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if summary.TotalStars != 12 {
			t.Errorf("expected 12 stars, got %d", summary.TotalStars)
		}
		if summary.CurrentStreakDays != 3 {
			t.Errorf("expected streak 3, got %d", summary.CurrentStreakDays)
		}
		if len(summary.CompletedLessonIDs) != 2 {
			t.Errorf("expected 2 completed lessons, got %d", len(summary.CompletedLessonIDs))
		}
		if summary.CompletedLessonIDs[0] != "lesson-1" {
			t.Errorf("expected lesson-1, got %s", summary.CompletedLessonIDs[0])
		}
	})
}
