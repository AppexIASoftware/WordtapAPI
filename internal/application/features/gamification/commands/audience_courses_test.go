package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type audienceCoursesFake struct {
	courses []entities.Course
	err     error
}

func (f audienceCoursesFake) FindByAuthor(_ context.Context, teacherID string) ([]entities.Course, error) {
	if teacherID != "teacher-1" {
		return nil, errors.New("unexpected teacher")
	}
	return f.courses, f.err
}

func TestAudienceCourseAuthorization(t *testing.T) {
	tests := []struct {
		name      string
		ids       []string
		repo      audienceCoursesFake
		wantError bool
	}{
		{"owned", []string{"owned"}, audienceCoursesFake{courses: []entities.Course{{ID: "owned"}}}, false},
		{"foreign", []string{"foreign"}, audienceCoursesFake{courses: []entities.Course{{ID: "owned"}}}, true},
		{"mixed", []string{"owned", "foreign"}, audienceCoursesFake{courses: []entities.Course{{ID: "owned"}}}, true},
		{"missing", []string{"missing"}, audienceCoursesFake{}, true},
		{"repository failure", []string{"owned"}, audienceCoursesFake{err: errors.New("database error")}, true},
		{"empty", []string{}, audienceCoursesFake{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			createRepo := newMockTeacherGameRepo()
			_, err := commands.NewCreateTeacherGameHandler(createRepo, tt.repo).Handle(ctx, commands.CreateTeacherGameCommand{TeacherID: "teacher-1", Title: "Game", AssignedCourseIDs: tt.ids})
			if (err != nil) != tt.wantError || (createRepo.created == nil) != tt.wantError {
				t.Fatalf("create: error=%v created=%v", err, createRepo.created != nil)
			}
			updateRepo := newMockTeacherGameRepo()
			updateRepo.games = []entities.TeacherGame{{ID: "game", TeacherID: "teacher-1"}}
			_, err = commands.NewUpdateTeacherGameHandler(updateRepo, tt.repo).Handle(ctx, commands.UpdateTeacherGameCommand{GameID: "game", TeacherID: "teacher-1", AssignedCourseIDs: &tt.ids})
			if (err != nil) != tt.wantError || updateRepo.updated == tt.wantError {
				t.Fatalf("update: error=%v updated=%v", err, updateRepo.updated)
			}
		})
	}
}

func TestAudienceCourseAuthorizationFailsClosed(t *testing.T) {
	ctx := context.Background()
	ids := []string{"foreign"}
	repo := newMockTeacherGameRepo()
	if _, err := commands.NewCreateTeacherGameHandler(repo).Handle(ctx, commands.CreateTeacherGameCommand{TeacherID: "teacher-1", Title: "Game", AssignedCourseIDs: ids}); err == nil || repo.created != nil {
		t.Fatal("creation must fail when audience ownership cannot be checked")
	}
	stored := `["foreign"]`
	repo.games = []entities.TeacherGame{{ID: "game", TeacherID: "teacher-1", AssignedCourseIDs: &stored}}
	if _, err := commands.NewUpdateTeacherGameHandler(repo, audienceCoursesFake{}).Handle(ctx, commands.UpdateTeacherGameCommand{GameID: "game", TeacherID: "teacher-1"}); err == nil || repo.updated {
		t.Fatal("update must reject an existing unauthorized audience")
	}
}
