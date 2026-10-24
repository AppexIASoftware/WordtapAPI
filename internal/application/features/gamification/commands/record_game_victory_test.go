package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type mockStatsRepo struct {
	lastParams repositories.RecordProgressParams
}

func (m *mockStatsRepo) GetLearningStats(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	return nil, nil
}

func (m *mockStatsRepo) DeductUserLife(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	return nil, nil
}

func (m *mockStatsRepo) RefillUserLives(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	return nil, nil
}

func (m *mockStatsRepo) RecordGameVictory(ctx context.Context, params repositories.RecordProgressParams) (*repositories.RecordProgressResult, error) {
	m.lastParams = params
	return &repositories.RecordProgressResult{
		LessonCompleted:   params.LessonID != "",
		StarsEarned:       params.StarsEarned,
		TotalStars:        params.StarsEarned,
		TotalPoints:       params.StarsEarned * 10,
		CurrentStreakDays: 1,
		WordsLearned:      params.WordsPracticed,
		LessonsCompleted:  1,
		GamesCompleted:    1,
	}, nil
}

func (m *mockStatsRepo) GetCompletedLessonIDs(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}

func TestRecordGameVictoryHandler(t *testing.T) {
	repo := &mockStatsRepo{}
	handler := commands.NewRecordGameVictoryHandler(repo)

	t.Run("fails when user_id is empty", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), commands.RecordGameVictoryCommand{
			UserID: "",
		})
		if err == nil {
			t.Fatalf("expected error for empty user_id, got nil")
		}
	})

	t.Run("clamps stars earned between 0 and 3", func(t *testing.T) {
		res, err := handler.Handle(context.Background(), commands.RecordGameVictoryCommand{
			UserID:         "user-123",
			LessonID:       "lesson-abc",
			StarsEarned:    5,
			WordsPracticed: 6,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastParams.StarsEarned != 3 {
			t.Errorf("expected stars clamped to 3, got %d", repo.lastParams.StarsEarned)
		}
		if res.StarsEarned != 3 {
			t.Errorf("expected result stars 3, got %d", res.StarsEarned)
		}
		if repo.lastParams.WordsPracticed != 6 {
			t.Errorf("expected 6 words, got %d", repo.lastParams.WordsPracticed)
		}
		if !res.LessonCompleted {
			t.Errorf("expected lesson to be marked completed")
		}
	})

	t.Run("handles negative values cleanly", func(t *testing.T) {
		_, err := handler.Handle(context.Background(), commands.RecordGameVictoryCommand{
			UserID:         "user-123",
			StarsEarned:    -2,
			WordsPracticed: -4,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastParams.StarsEarned != 0 {
			t.Errorf("expected stars clamped to 0, got %d", repo.lastParams.StarsEarned)
		}
		if repo.lastParams.WordsPracticed != 0 {
			t.Errorf("expected words clamped to 0, got %d", repo.lastParams.WordsPracticed)
		}
	})
}
