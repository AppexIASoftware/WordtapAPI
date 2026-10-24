package gamification_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/gamification"
)

func TestProgressRouter(t *testing.T) {
	e := echo.New()
	v1 := e.Group("/api/v1")
	repo := &mockStatsRepo{
		completedLessonIDs: []string{"lesson-xyz"},
	}

	finishHandler := commands.NewRecordGameVictoryHandler(repo)
	summaryHandler := queries.NewGetUserProgressHandler(repo)
	router := gamification.NewProgressRouter(finishHandler, summaryHandler)

	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "test-student-456")
			return next(c)
		}
	}
	router.RegisterRoutes(v1, authMiddleware)

	t.Run("POST /api/v1/gamification/progress/finish saves victory", func(t *testing.T) {
		payload := map[string]any{
			"lesson_id":       "lesson-101",
			"stars_earned":    3,
			"words_practiced": 6,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gamification/progress/finish", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. body: %s", rec.Code, rec.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				LessonCompleted bool `json:"lesson_completed"`
				StarsEarned     int  `json:"stars_earned"`
				TotalStars      int  `json:"total_stars"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if !res.Success {
			t.Errorf("expected success true")
		}
		if res.Data.StarsEarned != 3 {
			t.Errorf("expected 3 stars, got %d", res.Data.StarsEarned)
		}
		if !res.Data.LessonCompleted {
			t.Errorf("expected lesson to be completed")
		}
	})

	t.Run("GET /api/v1/gamification/progress/summary returns summary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/gamification/progress/summary", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				CompletedLessonIDs []string `json:"completed_lesson_ids"`
				TotalStars         int      `json:"total_stars"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if len(res.Data.CompletedLessonIDs) == 0 {
			t.Errorf("expected at least 1 completed lesson")
		}
	})
}
