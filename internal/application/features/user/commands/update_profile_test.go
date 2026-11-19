package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type updateProfileRepoStub struct {
	repositories.UserRepository
	user    *entities.User
	updated *entities.User
}

func (s *updateProfileRepoStub) FindByID(_ context.Context, id string) (*entities.User, error) {
	if s.user == nil || s.user.ID != id {
		return nil, errors.New("not found")
	}
	return s.user, nil
}

func (s *updateProfileRepoStub) UpdateUser(_ context.Context, user *entities.User) error {
	s.updated = user
	return nil
}

func TestUpdateProfileHandler_UpdatesNameAndLevel(t *testing.T) {
	repo := &updateProfileRepoStub{
		user: &entities.User{
			ID:            "user-1",
			Email:         "student@gmail.com",
			Name:          "Old Name",
			LearningLevel: "beginner",
		},
	}

	handler := NewUpdateProfileHandler(repo)
	res, err := handler.Handle(context.Background(), UpdateProfileCommand{
		UserID:        "user-1",
		Name:          "  Junior Vega  ",
		LearningLevel: "intermediate",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Name != "Junior Vega" || res.LearningLevel != "intermediate" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if repo.updated == nil || repo.updated.Name != "Junior Vega" {
		t.Fatalf("expected UpdateUser to be called with updated name")
	}
}

func TestUpdateProfileHandler_RejectsInvalidInput(t *testing.T) {
	repo := &updateProfileRepoStub{
		user: &entities.User{ID: "user-1", Name: "Valid", LearningLevel: "beginner"},
	}
	handler := NewUpdateProfileHandler(repo)

	if _, err := handler.Handle(context.Background(), UpdateProfileCommand{
		UserID: "user-1",
		Name:   "A",
	}); !errors.Is(err, ErrInvalidProfileName) {
		t.Fatalf("expected ErrInvalidProfileName, got %v", err)
	}

	if _, err := handler.Handle(context.Background(), UpdateProfileCommand{
		UserID:        "user-1",
		Name:          "Valid Name",
		LearningLevel: "expert",
	}); !errors.Is(err, ErrInvalidProfileLevel) {
		t.Fatalf("expected ErrInvalidProfileLevel, got %v", err)
	}
}
