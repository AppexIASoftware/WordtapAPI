package commands_test

import (
	"context"
	"errors"
	"testing"

	lessoncommands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/lessontest"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestUpdateLessonGameActivityAuthorizationAndFields(t *testing.T) {
	fixture := lessontest.NewFixture()
	h := lessoncommands.NewUpdateLessonGameActivityHandler(fixture.Lessons, fixture.Courses, fixture.Games)

	t.Run("teacher cannot update activity of course owned by another", func(t *testing.T) {
		_, err := h.Handle(context.Background(), lessoncommands.UpdateLessonGameActivityCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-teacher",
			RequesterID:   "teacher-2",
			RequesterRole: entities.RoleInstructor,
			Title:         "Hacked",
		})
		if !errors.Is(err, lessoncommands.ErrLessonGameAccessDenied) {
			t.Fatalf("expected ErrLessonGameAccessDenied, got %v", err)
		}
	})

	t.Run("teacher updates activity successfully", func(t *testing.T) {
		newDesc := "New challenge rules"
		game, err := h.Handle(context.Background(), lessoncommands.UpdateLessonGameActivityCommand{
			LessonID:         "lesson-teacher",
			GameID:           "game-teacher",
			RequesterID:      "teacher-1",
			RequesterRole:    entities.RoleInstructor,
			BaseArchetype:    "memory",
			Title:            "Updated Game",
			Description:      &newDesc,
			TimeLimitSeconds: 60,
			AllowedMistakes:  0,
			RewardXP:         50,
			ItemCount:        10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if game.BaseArchetype != "memory" {
			t.Errorf("expected memory, got %s", game.BaseArchetype)
		}
		if game.Title != "Updated Game" {
			t.Errorf("expected Updated Game, got %s", game.Title)
		}
		if game.Description == nil || *game.Description != newDesc {
			t.Errorf("expected description %s, got %v", newDesc, game.Description)
		}
		if game.TimeLimitSeconds != 60 || game.AllowedMistakes != 0 || game.RewardXP != 50 || game.ItemCount != 10 {
			t.Errorf("unexpected game rules: %+v", game)
		}
	})

	t.Run("teacher cannot mutate foreign or global game through own lesson", func(t *testing.T) {
		_, err := h.Handle(context.Background(), lessoncommands.UpdateLessonGameActivityCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-other",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
			Title:         "Hijacked",
		})
		if !errors.Is(err, lessoncommands.ErrLessonGameAccessDenied) {
			t.Fatalf("expected ErrLessonGameAccessDenied for foreign game, got %v", err)
		}

		_, err = h.Handle(context.Background(), lessoncommands.UpdateLessonGameActivityCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-global",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
			Title:         "Global Hijacked",
		})
		if !errors.Is(err, lessoncommands.ErrLessonGameAccessDenied) {
			t.Fatalf("expected ErrLessonGameAccessDenied for global game, got %v", err)
		}
	})

	t.Run("requires active lesson-game link when LinkRepo is attached", func(t *testing.T) {
		hLinked := lessoncommands.NewUpdateLessonGameActivityHandler(fixture.Lessons, fixture.Courses, fixture.Games).
			WithLinkRepo(fixture.Links)

		_, err := hLinked.Handle(context.Background(), lessoncommands.UpdateLessonGameActivityCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-teacher",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
			Title:         "Not Linked Yet",
		})
		if !errors.Is(err, lessoncommands.ErrLessonGameLinkNotFound) {
			t.Fatalf("expected ErrLessonGameLinkNotFound when unlinked, got %v", err)
		}

		_ = fixture.Links.Create(context.Background(), &entities.LessonGame{
			LessonID:      "lesson-teacher",
			TeacherGameID: "game-teacher",
		})
		game, err := hLinked.Handle(context.Background(), lessoncommands.UpdateLessonGameActivityCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-teacher",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
			Title:         "Linked Update OK",
		})
		if err != nil || game == nil || game.Title != "Linked Update OK" {
			t.Fatalf("expected linked update success, got %+v, err=%v", game, err)
		}
	})
}
