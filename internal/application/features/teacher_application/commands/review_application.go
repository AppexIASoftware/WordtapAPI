package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ReviewApplicationCommand struct {
	ApplicationID   string
	ReviewerID      string
	ReviewerRole    entities.UserRole
	Action          string // "approve" or "reject"
	RejectionReason string
}

type ReviewApplicationHandler struct {
	appRepo repositories.TeacherApplicationRepository
}

func NewReviewApplicationHandler(appRepo repositories.TeacherApplicationRepository) *ReviewApplicationHandler {
	return &ReviewApplicationHandler{appRepo: appRepo}
}

func (h *ReviewApplicationHandler) Handle(ctx context.Context, cmd ReviewApplicationCommand) (*entities.TeacherApplication, error) {
	if cmd.ApplicationID == "" {
		return nil, errors.New("application_id is required")
	}
	if cmd.ReviewerRole != entities.RoleAdmin && cmd.ReviewerRole != entities.RoleModerator {
		return nil, errors.New("forbidden: only administrators or moderators can review applications")
	}

	app, err := h.appRepo.FindByID(ctx, cmd.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}
	action := strings.ToLower(strings.TrimSpace(cmd.Action))
	switch action {
	case "approve":
		if app.Status == entities.ApplicationStatusApproved {
			return nil, errors.New("application is already approved")
		}
		if err := h.appRepo.ApproveInTx(ctx, cmd.ApplicationID, cmd.ReviewerID); err != nil {
			return nil, fmt.Errorf("failed to approve application: %w", err)
		}
	case "reject":
		if app.Status == entities.ApplicationStatusApproved {
			return nil, errors.New("cannot reject an already approved application; suspend instead")
		}
		if err := h.appRepo.RejectInTx(ctx, cmd.ApplicationID, cmd.ReviewerID, strings.TrimSpace(cmd.RejectionReason)); err != nil {
			return nil, fmt.Errorf("failed to reject application: %w", err)
		}
	case "suspend":
		if app.Status != entities.ApplicationStatusApproved {
			return nil, errors.New("only approved applications can be suspended")
		}
		if err := h.appRepo.SuspendInTx(ctx, cmd.ApplicationID, cmd.ReviewerID, strings.TrimSpace(cmd.RejectionReason)); err != nil {
			return nil, fmt.Errorf("failed to suspend application: %w", err)
		}
	case "reactivate":
		if app.Status != entities.ApplicationStatusSuspended && app.Status != entities.ApplicationStatusRejected {
			return nil, errors.New("only suspended or rejected applications can be reactivated")
		}
		if err := h.appRepo.ReactivateInTx(ctx, cmd.ApplicationID, cmd.ReviewerID); err != nil {
			return nil, fmt.Errorf("failed to reactivate application: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid review action: '%s', expected 'approve', 'reject', 'suspend', or 'reactivate'", cmd.Action)
	}

	return h.appRepo.FindByID(ctx, cmd.ApplicationID)
}
