package commands_test

import (
	"context"
	"errors"
	"testing"

	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/lessontest"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestLinkLessonGameHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("admin links global games to a platform lesson in order", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		handler := fixture.LinkHandler()

		first, err := handler.Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if first.SortOrder != 0 {
			t.Errorf("expected sort order 0, got %d", first.SortOrder)
		}
		if !first.IsRequired {
			t.Error("expected the link to be required by default")
		}
		if first.TeacherGame == nil || first.TeacherGame.Title != "Actividad Global" {
			t.Error("expected the linked game to be preloaded")
		}

		second, err := handler.Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global-2",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if second.SortOrder != 1 {
			t.Errorf("expected sort order 1, got %d", second.SortOrder)
		}
	})

	t.Run("admin cannot link games to a teacher course", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		_, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-global",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if !errors.Is(err, lessonCmd.ErrLessonGameAccessDenied) {
			t.Fatalf("expected access denied, got %v", err)
		}
	})

	t.Run("teacher links own game to own lesson", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		link, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-teacher",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if link.TeacherGameID != "game-teacher" {
			t.Errorf("expected game-teacher, got %s", link.TeacherGameID)
		}
	})

	t.Run("teacher can link a global game to own lesson", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		_, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-global",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, tt := range []struct {
		name          string
		gameID        string
		lessonID      string
		requesterID   string
		requesterRole entities.UserRole
	}{
		{"archived platform game", "game-global", "lesson-teacher", "teacher-1", entities.RoleInstructor},
		{"archived teacher game", "game-teacher", "lesson-teacher", "teacher-1", entities.RoleInstructor},
	} {
		t.Run(tt.name+" cannot be linked", func(t *testing.T) {
			fixture := lessontest.NewFixture()
			fixture.Games.Games[tt.gameID].IsActive = false
			link, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
				LessonID:      tt.lessonID,
				GameID:        tt.gameID,
				RequesterID:   tt.requesterID,
				RequesterRole: tt.requesterRole,
			})
			if !errors.Is(err, lessonCmd.ErrLessonGameGameNotFound) || link != nil {
				t.Fatalf("expected game not found without a link, got link=%v err=%v", link, err)
			}
			if len(fixture.Links.Links) != 0 {
				t.Fatalf("expected no persisted links, got %d", len(fixture.Links.Links))
			}
		})
	}

	t.Run("owner type permits a platform game without the legacy global flag", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		fixture.Games.Games["game-global"].IsGlobal = false
		if _, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-global",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		}); err != nil {
			t.Fatalf("expected platform owner type to allow the link, got %v", err)
		}
	})

	t.Run("legacy global flag does not grant access to another teacher's game", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		fixture.Games.Games["game-other"].IsGlobal = true
		if _, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-other",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		}); !errors.Is(err, lessonCmd.ErrLessonGameGameNotFound) {
			t.Fatalf("expected game not found for another teacher's game, got %v", err)
		}
	})

	t.Run("game above the course level is rejected", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		_, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-advanced",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if !errors.Is(err, lessonCmd.ErrLessonGameLevelTooHigh) {
			t.Fatalf("expected level too high, got %v", err)
		}
	})

	t.Run("game with the same level is allowed without warning", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		link, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-teacher",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if link.LevelWarning != "" {
			t.Errorf("expected no level warning, got %q", link.LevelWarning)
		}
	})

	t.Run("game below the course level is allowed with a review warning", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		link, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-basic",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if link.LevelWarning == "" {
			t.Error("expected a review warning for a lower level game")
		}
	})

	t.Run("teacher cannot link another teacher's game", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		_, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-teacher",
			GameID:        "game-other",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if !errors.Is(err, lessonCmd.ErrLessonGameGameNotFound) {
			t.Fatalf("expected game not found, got %v", err)
		}
	})

	t.Run("teacher cannot link to a platform lesson", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		_, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-teacher",
			RequesterID:   "teacher-1",
			RequesterRole: entities.RoleInstructor,
		})
		if !errors.Is(err, lessonCmd.ErrLessonGameAccessDenied) {
			t.Fatalf("expected access denied, got %v", err)
		}
	})

	t.Run("duplicate link is rejected", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		handler := fixture.LinkHandler()
		cmd := lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		}
		if _, err := handler.Handle(ctx, cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := handler.Handle(ctx, cmd); !errors.Is(err, lessonCmd.ErrLessonGameDuplicate) {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})

	t.Run("negative sort order is rejected", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		_, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global",
			SortOrder:     lessontest.IntPtr(-1),
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if err == nil {
			t.Fatal("expected a validation error")
		}
	})
}

func TestUpdateAndUnlinkLessonGameHandlers(t *testing.T) {
	ctx := context.Background()

	t.Run("update changes order and required flag", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		if _, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fixture.Games.Games["game-global"].IsActive = false

		link, err := fixture.UpdateHandler().Handle(ctx, lessonCmd.UpdateLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global",
			SortOrder:     lessontest.IntPtr(5),
			IsRequired:    lessontest.BoolPtr(false),
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if link.SortOrder != 5 {
			t.Errorf("expected sort order 5, got %d", link.SortOrder)
		}
		if link.IsRequired {
			t.Error("expected the link to be optional")
		}
		if len(fixture.Links.Links) != 1 {
			t.Errorf("expected archived game to remain linked, got %d links", len(fixture.Links.Links))
		}
	})

	t.Run("unlink is idempotent", func(t *testing.T) {
		fixture := lessontest.NewFixture()
		if _, err := fixture.LinkHandler().Handle(ctx, lessonCmd.LinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		handler := fixture.UnlinkHandler()
		cmd := lessonCmd.UnlinkLessonGameCommand{
			LessonID:      "lesson-platform",
			GameID:        "game-global",
			RequesterID:   "admin-1",
			RequesterRole: entities.RoleAdmin,
		}
		if err := handler.Handle(ctx, cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := handler.Handle(ctx, cmd); err != nil {
			t.Fatalf("expected idempotent unlink, got %v", err)
		}
		if len(fixture.Links.Links) != 0 {
			t.Errorf("expected no links, got %d", len(fixture.Links.Links))
		}
	})
}
