package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrInvalidProfileName  = errors.New("el nombre es obligatorio y debe tener entre 2 y 60 caracteres")
	ErrInvalidProfileLevel = errors.New("nivel de aprendizaje no válido")
)

var validLearningLevels = map[string]bool{
	"beginner":     true,
	"intermediate": true,
	"advanced":     true,
}

type UpdateProfileCommand struct {
	UserID        string
	Name          string
	LearningLevel string
	AvatarURL     *string
}

type UpdateProfileHandler struct {
	userRepo repositories.UserRepository
}

func NewUpdateProfileHandler(userRepo repositories.UserRepository) *UpdateProfileHandler {
	return &UpdateProfileHandler{userRepo: userRepo}
}

func (h *UpdateProfileHandler) Handle(ctx context.Context, cmd UpdateProfileCommand) (*AuthUserDTO, error) {
	trimmedName := strings.TrimSpace(cmd.Name)
	if len([]rune(trimmedName)) < 2 || len([]rune(trimmedName)) > 60 {
		return nil, ErrInvalidProfileName
	}

	normalizedLevel := strings.ToLower(strings.TrimSpace(cmd.LearningLevel))
	if normalizedLevel != "" && !validLearningLevels[normalizedLevel] {
		return nil, ErrInvalidProfileLevel
	}

	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	user.Name = trimmedName
	if normalizedLevel != "" {
		user.LearningLevel = normalizedLevel
	}
	if cmd.AvatarURL != nil {
		trimmedAvatar := strings.TrimSpace(*cmd.AvatarURL)
		if trimmedAvatar == "" {
			user.AvatarURL = nil
		} else {
			user.AvatarURL = &trimmedAvatar
		}
	}

	if err := h.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	dto := ToAuthUserDTO(user)
	return &dto, nil
}
