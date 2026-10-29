package queries_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockQueryGameRepo struct {
	games []entities.TeacherGame
}

func (m *mockQueryGameRepo) Create(ctx context.Context, g *entities.TeacherGame) error {
	return nil
}

func (m *mockQueryGameRepo) FindByID(ctx context.Context, id string) (*entities.TeacherGame, error) {
	for _, g := range m.games {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, nil
}

func (m *mockQueryGameRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.TeacherGame, error) {
	var res []entities.TeacherGame
	for _, g := range m.games {
		if g.TeacherID == teacherID {
			res = append(res, g)
		}
	}
	return res, nil
}

func (m *mockQueryGameRepo) FindAll(ctx context.Context) ([]entities.TeacherGame, error) {
	return m.games, nil
}

func (m *mockQueryGameRepo) FindGlobal(ctx context.Context) ([]entities.TeacherGame, error) {
	var res []entities.TeacherGame
	for _, g := range m.games {
		if g.OwnerType == entities.OwnerTypePlatform {
			res = append(res, g)
		}
	}
	return res, nil
}

func (m *mockQueryGameRepo) FindStandaloneByTeachers(ctx context.Context, teacherIDs []string) ([]entities.TeacherGame, error) {
	teacherSet := make(map[string]bool)
	for _, id := range teacherIDs {
		teacherSet[id] = true
	}
	var res []entities.TeacherGame
	for _, g := range m.games {
		if teacherSet[g.TeacherID] && g.IsActive && g.IsPublishedStandalone {
			res = append(res, g)
		}
	}
	return res, nil
}

func (m *mockQueryGameRepo) Update(ctx context.Context, g *entities.TeacherGame) error {
	return nil
}

func (m *mockQueryGameRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockQueryGameRepo) CountLessonUsage(ctx context.Context, gameID string) (int, error) {
	return 0, nil
}

func TestTeacherGameQueries(t *testing.T) {
	ctx := context.Background()
	repo := &mockQueryGameRepo{
		games: []entities.TeacherGame{
			{ID: "g-1", TeacherID: "teacher-1", Title: "Juego A"},
			{ID: "g-2", TeacherID: "teacher-1", Title: "Juego B"},
			{ID: "g-3", TeacherID: "teacher-2", Title: "Juego C"},
		},
	}

	t.Run("ListTeacherGamesHandler lista solo juegos del docente", func(t *testing.T) {
		handler := queries.NewListTeacherGamesHandler(repo)
		res, err := handler.Handle(ctx, queries.ListTeacherGamesQuery{TeacherID: "teacher-1"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(res) != 2 {
			t.Errorf("se esperaban 2 juegos, se obtuvieron %d", len(res))
		}
	})

	t.Run("GetTeacherGameDetailHandler obtiene detalle", func(t *testing.T) {
		handler := queries.NewGetTeacherGameDetailHandler(repo)
		res, err := handler.Handle(ctx, queries.GetTeacherGameDetailQuery{
			GameID:    "g-1",
			TeacherID: "teacher-1",
		})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if res.Title != "Juego A" {
			t.Errorf("título esperado 'Juego A', obtenido '%s'", res.Title)
		}
	})
}
