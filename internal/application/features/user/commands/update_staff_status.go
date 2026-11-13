package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrCannotSuspendSelf = errors.New("cannot change your own active status")
	ErrTargetUserMissing = errors.New("target user id is required")
)

type UpdateStaffStatusCommand struct {
	RequesterID  string
	TargetUserID string
	IsActive     bool
}

type UpdateStaffStatusHandler struct {
	userRepo repositories.StaffRepository
}

func NewUpdateStaffStatusHandler(userRepo repositories.StaffRepository) *UpdateStaffStatusHandler {
	return &UpdateStaffStatusHandler{userRepo: userRepo}
}

func (h *UpdateStaffStatusHandler) Handle(ctx context.Context, cmd UpdateStaffStatusCommand) (*queries.StaffUserDTO, error) {
	targetID := strings.TrimSpace(cmd.TargetUserID)
	if targetID == "" {
		return nil, ErrTargetUserMissing
	}
	if cmd.RequesterID == targetID && !cmd.IsActive {
		return nil, ErrCannotSuspendSelf
	}

	user, err := h.userRepo.FindByID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	user.IsActive = cmd.IsActive
	if err := h.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user status: %w", err)
	}

	if !cmd.IsActive {
		_ = h.userRepo.RevokeAllSessions(ctx, user.ID)
	}

	dto := queries.ToStaffUserDTO(user)
	return &dto, nil
}
