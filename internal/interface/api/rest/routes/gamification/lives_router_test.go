package gamification_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/gamification"
)

type mockStatsRepo struct {
	stats              *entities.UserLearningStats
	completedLessonIDs []string
}

func (m *mockStatsRepo) GetLearningStats(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	if m.stats == nil {
		m.stats = &entities.UserLearningStats{
			UserID:         userID,
			LivesRemaining: 5,
			MaxLives:       5,
		}
	}
	return m.stats, nil
}

func (m *mockStatsRepo) DeductUserLife(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	s, _ := m.GetLearningStats(ctx, userID)
	s.DeductLife()
	return s, nil
}

func (m *mockStatsRepo) RefillUserLives(ctx context.Context, userID string) (*entities.UserLearningStats, error) {
	s, _ := m.GetLearningStats(ctx, userID)
	s.RefillLives()
	return s, nil
}

func (m *mockStatsRepo) RecordGameVictory(ctx context.Context, params domainRepo.RecordProgressParams) (*domainRepo.RecordProgressResult, error) {
	s, _ := m.GetLearningStats(ctx, params.UserID)
	s.RecordVictory(params.StarsEarned, params.WordsPracticed, entities.UserLearningStats{}.UpdatedAt)
	if params.LessonID != "" {
		m.completedLessonIDs = append(m.completedLessonIDs, params.LessonID)
	}
	return &domainRepo.RecordProgressResult{
		LessonCompleted:   params.LessonID != "",
		StarsEarned:       params.StarsEarned,
		TotalStars:        s.TotalStars,
		TotalPoints:       s.TotalPoints,
		CurrentStreakDays: s.CurrentStreakDays,
		WordsLearned:      s.WordsLearned,
		LessonsCompleted:  s.LessonsCompleted,
		GamesCompleted:    s.GamesCompleted,
	}, nil
}

func (m *mockStatsRepo) GetCompletedLessonIDs(ctx context.Context, userID string) ([]string, error) {
	return m.completedLessonIDs, nil
}

func TestLivesRouter(t *testing.T) {
	e := echo.New()
	v1 := e.Group("/api/v1")
	repo := &mockStatsRepo{}
	router := gamification.NewLivesRouter(repo)

	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "test-user-123")
			return next(c)
		}
	}
	router.RegisterRoutes(v1, authMiddleware)

	t.Run("GET /api/v1/gamification/lives returns 5 hearts initial", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/gamification/lives", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				LivesRemaining int `json:"lives_remaining"`
				MaxLives       int `json:"max_lives"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if res.Data.LivesRemaining != 5 {
			t.Errorf("expected 5 lives, got %d", res.Data.LivesRemaining)
		}
	})

	t.Run("POST /api/v1/gamification/lives/deduct reduces 1 heart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gamification/lives/deduct", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var res struct {
			Data struct {
				LivesRemaining int `json:"lives_remaining"`
			} `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &res)
		if res.Data.LivesRemaining != 4 {
			t.Errorf("expected 4 lives, got %d", res.Data.LivesRemaining)
		}
	})

	t.Run("POST /api/v1/gamification/lives/refill restores hearts", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gamification/lives/refill", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var res struct {
			Data struct {
				LivesRemaining int `json:"lives_remaining"`
			} `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &res)
		if res.Data.LivesRemaining != 5 {
			t.Errorf("expected 5 lives, got %d", res.Data.LivesRemaining)
		}
	})
}
