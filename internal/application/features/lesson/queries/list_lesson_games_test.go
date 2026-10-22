package queries_test

import (
	"context"
	"errors"
	"testing"

	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/lessontest"
	lessonQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestListLessonGamesHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("owner sees active and inactive games", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		linkGames(t, fixture, "lesson-platform", "game-global", "admin-1", entities.RoleAdmin)
		linkGames(t, fixture, "lesson-platform", "game-global-2", "admin-1", entities.RoleAdmin)
		deactivateGame(fixture, "game-global-2")

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-platform",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos) != 2 {
			t.Fatalf("expected 2 games, got %d", len(dtos))
		}
	})

	t.Run("student sees only active games of a published lesson", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		linkGames(t, fixture, "lesson-teacher", "game-teacher", "teacher-1", entities.RoleInstructor)
		linkGames(t, fixture, "lesson-teacher", "game-global-2", "teacher-1", entities.RoleInstructor)
		deactivateGame(fixture, "game-global-2")

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-teacher",
			RequesterID:   "student-1",
			RequesterRole: entities.RoleStudent,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos) != 1 || dtos[0].GameID != "game-teacher" {
			t.Fatalf("expected only the active game, got %+v", dtos)
		}
	})

	t.Run("student cannot read a draft lesson", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		_, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-draft",
			RequesterID:   "student-1",
			RequesterRole: entities.RoleStudent,
		})
		if !errors.Is(err, lessonQuery.ErrLessonGamesAccessDenied) {
			t.Fatalf("expected access denied, got %v", err)
		}
	})

	t.Run("usage count reflects links across lessons", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		linkGames(t, fixture, "lesson-platform", "game-global", "admin-1", entities.RoleAdmin)
		linkGames(t, fixture, "lesson-teacher", "game-global", "teacher-1", entities.RoleInstructor)

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-platform",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos) != 1 {
			t.Fatalf("expected 1 game, got %d", len(dtos))
		}
		if dtos[0].UsageCount != 2 {
			t.Errorf("expected usage_count 2, got %d", dtos[0].UsageCount)
		}
		if dtos[0].LevelWarning != "" {
			t.Errorf("expected no level warning for an equal level, got %q", dtos[0].LevelWarning)
		}
	})

	t.Run("lower level game exposes a review warning", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		linkGames(t, fixture, "lesson-teacher", "game-basic", "teacher-1", entities.RoleInstructor)

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-teacher",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos) != 1 {
			t.Fatalf("expected 1 game, got %d", len(dtos))
		}
		if dtos[0].LevelWarning == "" {
			t.Error("expected a review warning for a lower level game")
		}
	})

	t.Run("words are derived from lesson cards", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		fixture.Lessons.Lessons["lesson-teacher"].Items = []entities.LessonItem{
			{
				ID: "item-vocab-1",
				VocabularyItem: &entities.VocabularyItem{
					ID:      "vocab-1",
					English: "Water",
					Spanish: "Agua",
				},
			},
			{
				ID: "item-phrase-1",
				Phrase: &entities.Phrase{
					ID:      "phrase-1",
					English: "How are you?",
					Spanish: "¿Cómo estás?",
				},
			},
			{
				ID: "item-vault-1",
				ContentBankItem: &entities.ContentBankItem{
					ID:         "vault-1",
					TargetText: "Book",
					PromptText: "Libro",
				},
			},
		}

		linkGames(t, fixture, "lesson-teacher", "game-teacher", "teacher-1", entities.RoleInstructor)

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-teacher",
			RequesterID:   "student-1",
			RequesterRole: entities.RoleStudent,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos) != 1 {
			t.Fatalf("expected 1 game, got %d", len(dtos))
		}
		words := dtos[0].Words
		if len(words) != 3 {
			t.Fatalf("expected 3 words, got %d", len(words))
		}
		if words[0].Term != "Water" || words[0].Translation != "Agua" || words[0].English != "Water" || words[0].Spanish != "Agua" {
			t.Errorf("unexpected word 0: %+v", words[0])
		}
		if words[1].Term != "How are you?" || words[1].Translation != "¿Cómo estás?" {
			t.Errorf("unexpected word 1: %+v", words[1])
		}
		if words[2].Term != "Book" || words[2].Translation != "Libro" {
			t.Errorf("unexpected word 2: %+v", words[2])
		}
	})

	t.Run("words respect item count limit of the game", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		fixture.Lessons.Lessons["lesson-teacher"].Items = []entities.LessonItem{
			{
				ID:             "item-1",
				VocabularyItem: &entities.VocabularyItem{English: "One", Spanish: "Uno"},
			},
			{
				ID:             "item-2",
				VocabularyItem: &entities.VocabularyItem{English: "Two", Spanish: "Dos"},
			},
		}
		fixture.Games.Games["game-teacher"].ItemCount = 1

		linkGames(t, fixture, "lesson-teacher", "game-teacher", "teacher-1", entities.RoleInstructor)

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-teacher",
			RequesterID:   "student-1",
			RequesterRole: entities.RoleStudent,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos[0].Words) != 1 {
			t.Errorf("expected 1 word capped by ItemCount, got %d", len(dtos[0].Words))
		}
		if dtos[0].Words[0].Term != "One" {
			t.Errorf("expected first word 'One', got %q", dtos[0].Words[0].Term)
		}
	})

	t.Run("falls back to custom_words_json when lesson has no cards", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		customJSON := `[{"term":"Cat","translation":"Gato"},{"term":"Dog","translation":"Perro"}]`
		fixture.Games.Games["game-teacher"].CustomWordsJSON = &customJSON

		linkGames(t, fixture, "lesson-teacher", "game-teacher", "teacher-1", entities.RoleInstructor)

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-teacher",
			RequesterID:   "student-1",
			RequesterRole: entities.RoleStudent,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos[0].Words) != 2 {
			t.Fatalf("expected 2 fallback words from json, got %d", len(dtos[0].Words))
		}
		if dtos[0].Words[0].Term != "Cat" || dtos[0].Words[0].Translation != "Gato" {
			t.Errorf("unexpected fallback word 0: %+v", dtos[0].Words[0])
		}
		if dtos[0].Words[1].Term != "Dog" || dtos[0].Words[1].Translation != "Perro" {
			t.Errorf("unexpected fallback word 1: %+v", dtos[0].Words[1])
		}
	})

	t.Run("words is empty slice not nil when no cards and no custom words", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		linkGames(t, fixture, "lesson-teacher", "game-teacher", "teacher-1", entities.RoleInstructor)

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-teacher",
			RequesterID:   "student-1",
			RequesterRole: entities.RoleStudent,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dtos[0].Words == nil {
			t.Error("expected words to be non-nil empty slice")
		}
		if len(dtos[0].Words) != 0 {
			t.Errorf("expected 0 words, got %d", len(dtos[0].Words))
		}
	})

	t.Run("extracts words from ContentText JSON cards created by studio", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		jsonCard := `{"en":"learn","es":"aprender","example":"I learn English"}`
		lesson := fixture.Lessons.Lessons["lesson-teacher"]
		lesson.Items = append(lesson.Items, entities.LessonItem{
			ID:          "card-json-1",
			LessonID:    "lesson-teacher",
			ContentText: &jsonCard,
		})
		fixture.Lessons.Lessons["lesson-teacher"] = lesson

		linkGames(t, fixture, "lesson-teacher", "game-teacher", "teacher-1", entities.RoleInstructor)

		dtos, err := fixture.ListHandler().Handle(ctx, lessonQuery.ListLessonGamesQuery{
			LessonID:      "lesson-teacher",
			RequesterID:   "student-1",
			RequesterRole: entities.RoleStudent,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dtos[0].Words) != 1 {
			t.Fatalf("expected 1 word, got %d", len(dtos[0].Words))
		}
		if dtos[0].Words[0].Term != "learn" || dtos[0].Words[0].Translation != "aprender" {
			t.Errorf("unexpected word from json: %+v", dtos[0].Words[0])
		}
	})
}

func deactivateGame(fixture *lessontest.Fixture, gameID string) {
	fixture.Games.Games[gameID].IsActive = false
}

func linkGames(
	t *testing.T,
	fixture *lessontest.Fixture,
	lessonID, gameID, requesterID string,
	role entities.UserRole,
) {
	t.Helper()
	if _, err := fixture.LinkHandler().Handle(context.Background(), lessonCmd.LinkLessonGameCommand{
		LessonID:      lessonID,
		GameID:        gameID,
		RequesterID:   requesterID,
		RequesterRole: role,
	}); err != nil {
		t.Fatalf("unexpected link error: %v", err)
	}
}
