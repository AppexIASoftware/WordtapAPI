package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockTeacherGameRepo struct {
	games   []entities.TeacherGame
	created *entities.TeacherGame
	updated bool
	deleted bool
}

func newMockTeacherGameRepo() *mockTeacherGameRepo {
	return &mockTeacherGameRepo{
		games: make([]entities.TeacherGame, 0),
	}
}

func (m *mockTeacherGameRepo) Create(ctx context.Context, g *entities.TeacherGame) error {
	m.created = g
	m.games = append(m.games, *g)
	return nil
}

func (m *mockTeacherGameRepo) FindByID(ctx context.Context, id string) (*entities.TeacherGame, error) {
	for _, g := range m.games {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, nil
}

func (m *mockTeacherGameRepo) FindByTeacher(ctx context.Context, teacherID string) ([]entities.TeacherGame, error) {
	var res []entities.TeacherGame
	for _, g := range m.games {
		if g.TeacherID == teacherID {
			res = append(res, g)
		}
	}
	return res, nil
}

func (m *mockTeacherGameRepo) FindAll(ctx context.Context) ([]entities.TeacherGame, error) {
	return m.games, nil
}

func (m *mockTeacherGameRepo) Update(ctx context.Context, g *entities.TeacherGame) error {
	for i, existing := range m.games {
		if existing.ID == g.ID {
			m.games[i] = *g
			m.updated = true
			return nil
		}
	}
	return nil
}

func (m *mockTeacherGameRepo) Delete(ctx context.Context, id string) error {
	for i, existing := range m.games {
		if existing.ID == id {
			m.games = append(m.games[:i], m.games[i+1:]...)
			m.deleted = true
			return nil
		}
	}
	return nil
}

func TestTeacherGameCommands(t *testing.T) {
	ctx := context.Background()
	repo := newMockTeacherGameRepo()

	t.Run("CreateTeacherGameHandler exitoso", func(t *testing.T) {
		handler := commands.NewCreateTeacherGameHandler(repo)
		cmd := commands.CreateTeacherGameCommand{
			TeacherID:        "teacher-1",
			Title:            "Desafío Verbos Irregulares",
			BaseArchetype:    "match",
			Level:            "B1 Intermediate",
			TimeLimitSeconds: 30,
			RewardXP:         25,
			ItemCount:        8,
		}

		res, err := handler.Handle(ctx, cmd)
		if err != nil {
			t.Fatalf("error creando juego: %v", err)
		}
		if res.Title != "Desafío Verbos Irregulares" {
			t.Errorf("título esperado 'Desafío Verbos Irregulares', obtenido '%s'", res.Title)
		}
		if res.BaseArchetype != "match" {
			t.Errorf("arquetipo esperado 'match', obtenido '%s'", res.BaseArchetype)
		}
	})

	t.Run("UpdateTeacherGameHandler exitoso", func(t *testing.T) {
		repo.games = append(repo.games, entities.TeacherGame{
			ID:               "game-10",
			TeacherID:        "teacher-1",
			Title:            "Juego Original",
			BaseArchetype:    "memory",
			TimeLimitSeconds: 45,
		})

		handler := commands.NewUpdateTeacherGameHandler(repo)
		cmd := commands.UpdateTeacherGameCommand{
			GameID:           "game-10",
			TeacherID:        "teacher-1",
			Title:            "Juego Actualizado",
			TimeLimitSeconds: 60,
		}

		res, err := handler.Handle(ctx, cmd)
		if err != nil {
			t.Fatalf("error actualizando juego: %v", err)
		}
		if res.Title != "Juego Actualizado" {
			t.Errorf("título esperado 'Juego Actualizado', obtenido '%s'", res.Title)
		}
		if res.TimeLimitSeconds != 60 {
			t.Errorf("tiempo esperado 60, obtenido %d", res.TimeLimitSeconds)
		}
	})

	t.Run("DeleteTeacherGameHandler exitoso", func(t *testing.T) {
		repo.games = append(repo.games, entities.TeacherGame{
			ID:        "game-20",
			TeacherID: "teacher-1",
			Title:     "Para Eliminar",
		})

		handler := commands.NewDeleteTeacherGameHandler(repo)
		err := handler.Handle(ctx, commands.DeleteTeacherGameCommand{
			GameID:    "game-20",
			TeacherID: "teacher-1",
		})
		if err != nil {
			t.Fatalf("error eliminando juego: %v", err)
		}
		if !repo.deleted {
			t.Errorf("se esperaba que el juego fuera eliminado")
		}
	})
}
