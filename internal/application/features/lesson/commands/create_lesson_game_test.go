package commands_test

import (
	"context"
	"errors"
	"testing"

	gamecommands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	lessoncommands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/lessontest"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestCreateLessonGameAuthorizationAndScope(t *testing.T) {
	for _, tc := range []struct {
		name, lesson, user string
		role               entities.UserRole
		owner              entities.OwnerType
		allowed            bool
	}{
		{"teacher owns course", "lesson-teacher", "teacher-1", entities.RoleInstructor, entities.OwnerTypeTeacher, true},
		{"admin owns platform course", "lesson-platform", "admin-1", entities.RoleAdmin, entities.OwnerTypePlatform, true},
		{"other teacher denied", "lesson-teacher", "teacher-2", entities.RoleInstructor, "", false},
		{"admin denied teacher course", "lesson-teacher", "admin-1", entities.RoleAdmin, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := lessontest.NewFixture()
			h := lessoncommands.NewCreateLessonGameHandler(f.Lessons, f.Courses, gamecommands.NewCreateTeacherGameHandler(f.Games))
			fakeID, fakeWords := "another-lesson", `[{"term":"leak","translation":"fuga"}]`
			game, err := h.Handle(context.Background(), lessoncommands.CreateLessonGameCommand{
				LessonID: tc.lesson, RequesterID: tc.user, RequesterRole: tc.role,
				Game: gamecommands.CreateTeacherGameCommand{Title: "Activity", SourceID: &fakeID, SourceType: "vault", CustomWordsJSON: &fakeWords},
			})
			if !tc.allowed {
				if !errors.Is(err, lessoncommands.ErrLessonGameAccessDenied) || game != nil {
					t.Fatalf("expected denied, got %v %v", game, err)
				}
				return
			}
			if err != nil || game == nil || game.IsGlobal || game.OwnerType != tc.owner || game.SourceID == nil || *game.SourceID != tc.lesson || game.SourceType != "course_lesson" || game.CustomWordsJSON != nil {
				t.Fatalf("unexpected game: %+v, err=%v", game, err)
			}
		})
	}
}
