package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrCannotDeleteSelf      = errors.New("No puedes eliminar tu propia cuenta.")
	ErrUserHasBillingRecords = errors.New("No se puede eliminar este usuario porque tiene registros asociados (cursos, revisiones u órdenes). Suspende su acceso en su lugar.")
)

type DeleteStaffCommand struct {
	RequesterID  string
	TargetUserID string
}

type DeleteStaffHandler struct {
	staffRepo repositories.StaffRepository
}

func NewDeleteStaffHandler(staffRepo repositories.StaffRepository) *DeleteStaffHandler {
	return &DeleteStaffHandler{staffRepo: staffRepo}
}

func (h *DeleteStaffHandler) Handle(ctx context.Context, cmd DeleteStaffCommand) error {
	targetID := strings.TrimSpace(cmd.TargetUserID)
	if targetID == "" {
		return ErrTargetUserMissing
	}
	if cmd.RequesterID == targetID {
		return ErrCannotDeleteSelf
	}

	user, err := h.staffRepo.FindByID(ctx, targetID)
	if err != nil {
		return err
	}

	_ = h.staffRepo.RevokeAllSessions(ctx, user.ID)
	if err := h.staffRepo.DeleteUser(ctx, user.ID); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "foreign key") || strings.Contains(msg, "constraint") || strings.Contains(msg, "1451") {
			return ErrUserHasBillingRecords
		}
		return fmt.Errorf("failed to delete staff user: %w", err)
	}

	return nil
}
