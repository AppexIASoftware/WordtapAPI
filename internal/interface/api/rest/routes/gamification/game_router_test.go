package gamification_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	gameCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	gameQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/gamification"
)

type mockGameRouterRepo struct {
	games []entities.TeacherGame
}

func newMockGameRouterRepo() *mockGameRouterRepo {
	return &mockGameRouterRepo{
		games: make([]entities.TeacherGame, 0),
	}
}

func (m *mockGameRouterRepo) Create(ctx context.Context, g *entities.TeacherGame) error {
	m.games = append(m.games, *g)
	return nil
}

func (m *mockGameRouterRepo) FindByID(ctx context.Context, id string) (*entities.TeacherGame, error) {
	for _, g := range m.games {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, nil
}

func (m *mockGameRouterRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.TeacherGame, error) {
	var res []entities.TeacherGame
	for _, g := range m.games {
		if g.TeacherID == teacherID {
			res = append(res, g)
		}
	}
	return res, nil
}

func (m *mockGameRouterRepo) Update(ctx context.Context, g *entities.TeacherGame) error {
	for i, existing := range m.games {
		if existing.ID == g.ID {
			m.games[i] = *g
			return nil
		}
	}
	return nil
}

func (m *mockGameRouterRepo) Delete(ctx context.Context, id string) error {
	for i, existing := range m.games {
		if existing.ID == id {
			m.games = append(m.games[:i], m.games[i+1:]...)
			return nil
		}
	}
	return nil
}

func setupGameRouter() (*echo.Echo, *mockGameRouterRepo) {
	e := echo.New()
	repo := newMockGameRouterRepo()

	createHandler := gameCmd.NewCreateTeacherGameHandler(repo)
	updateHandler := gameCmd.NewUpdateTeacherGameHandler(repo)
	deleteHandler := gameCmd.NewDeleteTeacherGameHandler(repo)
	listHandler := gameQuery.NewListTeacherGamesHandler(repo)
	getHandler := gameQuery.NewGetTeacherGameDetailHandler(repo)

	r := gamification.NewGameRouter(createHandler, updateHandler, deleteHandler, listHandler, getHandler)

	v1 := e.Group("/api/v1")
	passMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-uuid-1")
			return next(c)
		}
	}
	r.RegisterRoutes(v1, passMiddleware, passMiddleware)

	return e, repo
}

func TestGameRouter_Endpoints(t *testing.T) {
	e, repo := setupGameRouter()

	t.Run("POST /api/v1/teacher/games crea actividad", func(t *testing.T) {
		payload := map[string]any{
			"title":              "WordMatch Vocabulario",
			"base_archetype":     "match",
			"time_limit_seconds": 30,
			"reward_xp":          20,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/games", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("código esperado %d, obtenido %d", http.StatusCreated, rec.Code)
		}
	})

	t.Run("GET /api/v1/teacher/games lista actividades", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/games", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código esperado %d, obtenido %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("PUT /api/v1/teacher/games/:id actualiza actividad", func(t *testing.T) {
		_ = repo.Create(context.Background(), &entities.TeacherGame{
			ID:        "game-test-1",
			TeacherID: "teacher-uuid-1",
			Title:     "Juego A",
		})

		updatePayload := map[string]any{
			"title":              "Juego Actualizado",
			"time_limit_seconds": 50,
		}
		body, _ := json.Marshal(updatePayload)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/teacher/games/game-test-1", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código esperado %d, obtenido %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("DELETE /api/v1/teacher/games/:id elimina actividad", func(t *testing.T) {
		_ = repo.Create(context.Background(), &entities.TeacherGame{
			ID:        "game-test-2",
			TeacherID: "teacher-uuid-1",
			Title:     "Juego Para Borrar",
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/teacher/games/game-test-2", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("código esperado %d, obtenido %d", http.StatusOK, rec.Code)
		}
	})
}
