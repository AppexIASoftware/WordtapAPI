package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestUpdateTeacherProfileHandler_UpdatesBioAndSpecialty(t *testing.T) {
	repo := &updateProfileRepoStub{
		user: &entities.User{
			ID:    "teacher-1",
			Email: "teacher@wordtap.app",
			Name:  "Prof. Ana",
			Role:  entities.RoleInstructor,
		},
	}

	bio := "  Licenciada en Lenguas Modernas con 8 años de experiencia.  "
	specialty := "  Inglés de Negocios y Fonética  "

	handler := NewUpdateTeacherProfileHandler(repo)
	res, err := handler.Handle(context.Background(), UpdateTeacherProfileCommand{
		UserID:    "teacher-1",
		Bio:       &bio,
		Specialty: &specialty,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Bio == nil || *res.Bio != "Licenciada en Lenguas Modernas con 8 años de experiencia." {
		t.Fatalf("unexpected bio: %v", res.Bio)
	}
	if res.Specialty == nil || *res.Specialty != "Inglés de Negocios y Fonética" {
		t.Fatalf("unexpected specialty: %v", res.Specialty)
	}
}

func TestUpdateTeacherProfileHandler_ClearsEmptyFields(t *testing.T) {
	oldBio := "Bio previa"
	oldSpec := "Inglés"
	repo := &updateProfileRepoStub{
		user: &entities.User{
			ID:        "teacher-1",
			Name:      "Prof. Ana",
			Bio:       &oldBio,
			Specialty: &oldSpec,
		},
	}

	empty := "   "
	handler := NewUpdateTeacherProfileHandler(repo)
	res, err := handler.Handle(context.Background(), UpdateTeacherProfileCommand{
		UserID:    "teacher-1",
		Bio:       &empty,
		Specialty: &empty,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Bio != nil || res.Specialty != nil {
		t.Fatalf("expected nil bio and specialty, got bio=%v specialty=%v", res.Bio, res.Specialty)
	}
}

func TestUpdateTeacherProfileHandler_RejectsOversizedFields(t *testing.T) {
	repo := &updateProfileRepoStub{
		user: &entities.User{ID: "teacher-1", Name: "Prof. Ana"},
	}
	handler := NewUpdateTeacherProfileHandler(repo)

	tooLongBio := strings.Repeat("á", MaxTeacherBioLength+1)
	if _, err := handler.Handle(context.Background(), UpdateTeacherProfileCommand{
		UserID: "teacher-1",
		Bio:    &tooLongBio,
	}); !errors.Is(err, ErrInvalidTeacherBio) {
		t.Fatalf("expected ErrInvalidTeacherBio, got %v", err)
	}

	tooLongSpec := strings.Repeat("b", MaxTeacherSpecialtyLength+1)
	if _, err := handler.Handle(context.Background(), UpdateTeacherProfileCommand{
		UserID:    "teacher-1",
		Specialty: &tooLongSpec,
	}); !errors.Is(err, ErrInvalidTeacherSpecialty) {
		t.Fatalf("expected ErrInvalidTeacherSpecialty, got %v", err)
	}
}
