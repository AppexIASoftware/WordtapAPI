package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockTeacherGameRepo struct {
	games   []entities.TeacherGame
	usage   map[string]int
	created *entities.TeacherGame
	updated bool
	deleted bool
}

func newMockTeacherGameRepo() *mockTeacherGameRepo {
	return &mockTeacherGameRepo{
		games: make([]entities.TeacherGame, 0),
		usage: make(map[string]int),
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

func (m *mockTeacherGameRepo) FindGlobal(ctx context.Context) ([]entities.TeacherGame, error) {
	var res []entities.TeacherGame
	for _, g := range m.games {
		if g.OwnerType == entities.OwnerTypePlatform {
			res = append(res, g)
		}
	}
	return res, nil
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

func (m *mockTeacherGameRepo) CountLessonUsage(ctx context.Context, gameID string) (int, error) {
	return m.usage[gameID], nil
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
		if res.OwnerType != entities.OwnerTypeTeacher {
			t.Errorf("propiedad esperada 'teacher', obtenida '%s'", res.OwnerType)
		}
		if res.IsGlobal {
			t.Error("una actividad de docente no puede ser global")
		}
	})

	t.Run("CreateGlobalGameHandler asigna propiedad de plataforma", func(t *testing.T) {
		handler := commands.NewCreateGlobalGameHandler(repo)
		res, err := handler.Handle(ctx, commands.CreateGlobalGameCommand{
			AuthorID:      "admin-1",
			Title:         "Actividad Global",
			BaseArchetype: "memory",
		})
		if err != nil {
			t.Fatalf("error creando actividad global: %v", err)
		}
		if res.OwnerType != entities.OwnerTypePlatform {
			t.Errorf("propiedad esperada 'platform', obtenida '%s'", res.OwnerType)
		}
		if !res.IsGlobal {
			t.Error("una actividad de plataforma debe ser global")
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

	t.Run("docente no puede editar una actividad global", func(t *testing.T) {
		repo.games = append(repo.games, entities.TeacherGame{
			ID:        "game-global",
			TeacherID: "admin-1",
			Title:     "Actividad Global",
			OwnerType: entities.OwnerTypePlatform,
			IsGlobal:  true,
		})

		handler := commands.NewUpdateTeacherGameHandler(repo)
		_, err := handler.Handle(ctx, commands.UpdateTeacherGameCommand{
			GameID:    "game-global",
			TeacherID: "teacher-1",
			Title:     "Intento de edición",
		})
		if err == nil {
			t.Fatal("se esperaba error al editar una actividad global")
		}
	})

	t.Run("docente no puede eliminar una actividad global", func(t *testing.T) {
		repo.games = append(repo.games, entities.TeacherGame{
			ID:        "game-global-2",
			TeacherID: "admin-1",
			Title:     "Actividad Global",
			OwnerType: entities.OwnerTypePlatform,
			IsGlobal:  true,
		})

		handler := commands.NewDeleteTeacherGameHandler(repo)
		err := handler.Handle(ctx, commands.DeleteTeacherGameCommand{
			GameID:    "game-global-2",
			TeacherID: "teacher-1",
		})
		if err == nil {
			t.Fatal("se esperaba error al eliminar una actividad global")
		}
	})

	t.Run("DeleteTeacherGameHandler archiva una actividad en uso", func(t *testing.T) {
		repo.games = append(repo.games, entities.TeacherGame{
			ID:        "game-30",
			TeacherID: "teacher-1",
			Title:     "Actividad En Uso",
			IsActive:  true,
		})
		repo.usage["game-30"] = 2

		handler := commands.NewDeleteTeacherGameHandler(repo)
		if err := handler.Handle(ctx, commands.DeleteTeacherGameCommand{
			GameID:    "game-30",
			TeacherID: "teacher-1",
		}); err != nil {
			t.Fatalf("error archivando juego: %v", err)
		}

		stored, _ := repo.FindByID(ctx, "game-30")
		if stored == nil {
			t.Fatal("una actividad en uso debe conservarse archivada")
		}
		if stored.IsActive {
			t.Error("una actividad en uso debe quedar inactiva")
		}
	})

	t.Run("DeleteGlobalGameHandler archiva una actividad global en uso", func(t *testing.T) {
		repo.games = append(repo.games, entities.TeacherGame{
			ID:        "game-global-3",
			TeacherID: "admin-1",
			Title:     "Global En Uso",
			OwnerType: entities.OwnerTypePlatform,
			IsGlobal:  true,
			IsActive:  true,
		})
		repo.usage["game-global-3"] = 1

		handler := commands.NewDeleteGlobalGameHandler(repo)
		res, err := handler.Handle(ctx, commands.DeleteGlobalGameCommand{GameID: "game-global-3"})
		if err != nil {
			t.Fatalf("error archivando actividad global: %v", err)
		}
		if !res.Archived || res.UsageCount != 1 {
			t.Errorf("se esperaba archived=true y usage_count=1, obtenido %+v", res)
		}

		stored, _ := repo.FindByID(ctx, "game-global-3")
		if stored == nil || stored.IsActive {
			t.Errorf("la actividad global debe quedar archivada, obtenido %+v", stored)
		}
	})
}
