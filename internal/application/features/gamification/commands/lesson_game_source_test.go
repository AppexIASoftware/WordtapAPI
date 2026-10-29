package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestUpdateTeacherGameKeepsLessonSource(t *testing.T) {
	for _, tc := range []struct {
		name, sourceType, sourceID string
		wantError                  bool
	}{
		{"same lesson", "course_lesson", "lesson-1", false},
		{"other lesson", "course_lesson", "lesson-2", true},
		{"other source type", "vault", "lesson-1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "lesson-1"
			repo := newMockTeacherGameRepo()
			repo.games = []entities.TeacherGame{{ID: "game", TeacherID: "author", SourceType: "course_lesson", SourceID: &source}}
			_, err := commands.NewUpdateTeacherGameHandler(repo).Handle(context.Background(), commands.UpdateTeacherGameCommand{
				GameID: "game", TeacherID: "author", SourceType: tc.sourceType, SourceID: &tc.sourceID,
			})
			if (err != nil) != tc.wantError || repo.updated == tc.wantError {
				t.Fatalf("error=%v updated=%v", err, repo.updated)
			}
		})
	}
}

func TestUpdatePlatformGameKeepsLessonSource(t *testing.T) {
	source, other := "lesson-1", "lesson-2"
	repo := newMockTeacherGameRepo()
	repo.games = []entities.TeacherGame{{ID: "game", OwnerType: entities.OwnerTypePlatform, SourceType: "course_lesson", SourceID: &source}}
	_, err := commands.NewUpdateGlobalGameHandler(repo).Handle(context.Background(), commands.UpdateGlobalGameCommand{
		GameID: "game", AuthorID: "admin", SourceID: &other,
	})
	if err == nil || repo.updated {
		t.Fatalf("expected rejection without update, err=%v updated=%v", err, repo.updated)
	}
}
