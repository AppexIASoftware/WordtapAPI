package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type DeleteApplicationCommand struct {
	ApplicationID string
	ReviewerID    string
	ReviewerRole  entities.UserRole
}

type DeleteApplicationHandler struct {
	appRepo repositories.TeacherApplicationRepository
}

func NewDeleteApplicationHandler(appRepo repositories.TeacherApplicationRepository) *DeleteApplicationHandler {
	return &DeleteApplicationHandler{appRepo: appRepo}
}

func (h *DeleteApplicationHandler) Handle(ctx context.Context, cmd DeleteApplicationCommand) error {
	if cmd.ApplicationID == "" {
		return errors.New("application_id is required")
	}
	if cmd.ReviewerRole != entities.RoleAdmin && cmd.ReviewerRole != entities.RoleModerator {
		return errors.New("forbidden: only administrators or moderators can delete applications")
	}

	if _, err := h.appRepo.FindByID(ctx, cmd.ApplicationID); err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	return h.appRepo.DeleteInTx(ctx, cmd.ApplicationID)
}
