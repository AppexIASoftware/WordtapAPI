package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

const (
	MaxTeacherBioLength       = 500
	MaxTeacherSpecialtyLength = 120
)

var (
	ErrInvalidTeacherBio       = errors.New("la biografía académica no puede superar los 500 caracteres")
	ErrInvalidTeacherSpecialty = errors.New("la especialidad académica no puede superar los 120 caracteres")
)

type UpdateTeacherProfileCommand struct {
	UserID    string
	Bio       *string
	Specialty *string
}

type UpdateTeacherProfileHandler struct {
	userRepo repositories.UserRepository
}

func NewUpdateTeacherProfileHandler(userRepo repositories.UserRepository) *UpdateTeacherProfileHandler {
	return &UpdateTeacherProfileHandler{userRepo: userRepo}
}

func (h *UpdateTeacherProfileHandler) Handle(ctx context.Context, cmd UpdateTeacherProfileCommand) (*AuthUserDTO, error) {
	var normalizedBio *string
	if cmd.Bio != nil {
		trimmed := strings.TrimSpace(*cmd.Bio)
		if len([]rune(trimmed)) > MaxTeacherBioLength {
			return nil, ErrInvalidTeacherBio
		}
		if trimmed != "" {
			normalizedBio = &trimmed
		}
	}

	var normalizedSpecialty *string
	if cmd.Specialty != nil {
		trimmed := strings.TrimSpace(*cmd.Specialty)
		if len([]rune(trimmed)) > MaxTeacherSpecialtyLength {
			return nil, ErrInvalidTeacherSpecialty
		}
		if trimmed != "" {
			normalizedSpecialty = &trimmed
		}
	}

	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	if cmd.Bio != nil {
		user.Bio = normalizedBio
	}
	if cmd.Specialty != nil {
		user.Specialty = normalizedSpecialty
	}

	if err := h.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	dto := ToAuthUserDTO(user)
	return &dto, nil
}
