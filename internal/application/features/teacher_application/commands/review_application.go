package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
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

const defaultAutoApplicationBio = "Postulante registrado vía Google en Wordtap Studio"

func hasCompleteAcademicProfile(app *entities.TeacherApplication) bool {
	if app.Specialty == nil || strings.TrimSpace(*app.Specialty) == "" {
		return false
	}
	if app.Bio == nil {
		return false
	}
	bio := strings.TrimSpace(*app.Bio)
	if bio == defaultAutoApplicationBio || len([]rune(bio)) < 20 {
		return false
	}
	return true
}

func (h *ReviewApplicationHandler) Handle(ctx context.Context, cmd ReviewApplicationCommand) (*entities.TeacherApplication, error) {
	if cmd.ApplicationID == "" {
		return nil, errors.New("application_id is required")
	}
	if cmd.ReviewerRole != entities.RoleAdmin && cmd.ReviewerRole != entities.RoleModerator {
		return nil, apperrors.New("No tienes permisos para revisar postulaciones.")
	}

	app, err := h.appRepo.FindByID(ctx, cmd.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}
	action := strings.ToLower(strings.TrimSpace(cmd.Action))
	switch action {
	case "approve":
		if app.Status == entities.ApplicationStatusApproved {
			return nil, apperrors.New("La postulación ya se encuentra aprobada.")
		}
		if !hasCompleteAcademicProfile(app) {
			return nil, apperrors.New("El postulante debe completar su especialidad académica y una biografía de al menos 20 caracteres antes de ser aprobado.")
		}
		if err := h.appRepo.ApproveInTx(ctx, cmd.ApplicationID, cmd.ReviewerID); err != nil {
			return nil, fmt.Errorf("failed to approve application: %w", err)
		}
	case "reject":
		if app.Status == entities.ApplicationStatusApproved {
			return nil, apperrors.New("No se puede rechazar una postulación ya aprobada. Puedes suspenderla en su lugar.")
		}
		if err := h.appRepo.RejectInTx(ctx, cmd.ApplicationID, cmd.ReviewerID, strings.TrimSpace(cmd.RejectionReason)); err != nil {
			return nil, fmt.Errorf("failed to reject application: %w", err)
		}
	case "suspend":
		if app.Status != entities.ApplicationStatusApproved {
			return nil, apperrors.New("Solo las postulaciones aprobadas pueden suspenderse.")
		}
		if err := h.appRepo.SuspendInTx(ctx, cmd.ApplicationID, cmd.ReviewerID, strings.TrimSpace(cmd.RejectionReason)); err != nil {
			return nil, fmt.Errorf("failed to suspend application: %w", err)
		}
	case "reactivate":
		if app.Status != entities.ApplicationStatusSuspended && app.Status != entities.ApplicationStatusRejected {
			return nil, apperrors.New("Solo las postulaciones suspendidas o rechazadas pueden reactivarse.")
		}
		if err := h.appRepo.ReactivateInTx(ctx, cmd.ApplicationID, cmd.ReviewerID); err != nil {
			return nil, fmt.Errorf("failed to reactivate application: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid review action: '%s', expected 'approve', 'reject', 'suspend', or 'reactivate'", cmd.Action)
	}

	return h.appRepo.FindByID(ctx, cmd.ApplicationID)
}
