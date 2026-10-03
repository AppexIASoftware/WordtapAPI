package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ApplyTeacherCommand struct {
	UserID    string
	Bio       *string
	Specialty *string
}

type ApplyTeacherHandler struct {
	appRepo  repositories.TeacherApplicationRepository
	userRepo repositories.UserRepository
}

func NewApplyTeacherHandler(
	appRepo repositories.TeacherApplicationRepository,
	userRepo repositories.UserRepository,
) *ApplyTeacherHandler {
	return &ApplyTeacherHandler{
		appRepo:  appRepo,
		userRepo: userRepo,
	}
}

func (h *ApplyTeacherHandler) Handle(ctx context.Context, cmd ApplyTeacherCommand) (*entities.TeacherApplication, error) {
	if cmd.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.Role == entities.RoleInstructor || user.Role == entities.RoleAdmin {
		return nil, errors.New("user is already an authorized instructor or admin")
	}

	// Verificar si ya tiene postulación previa
	existing, err := h.appRepo.FindByUserID(ctx, cmd.UserID)
	if err == nil && existing != nil {
		if existing.Status == entities.ApplicationStatusPending {
			return existing, nil
		}
		if existing.Status == entities.ApplicationStatusApproved {
			return existing, nil
		}
		// Si fue rechazada, permitimos re-postularse
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed checking previous applications: %w", err)
	}

	now := time.Now()
	newApp := &entities.TeacherApplication{
		UserID:    cmd.UserID,
		Status:    entities.ApplicationStatusPending,
		Bio:       cmd.Bio,
		Specialty: cmd.Specialty,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.appRepo.Create(ctx, newApp); err != nil {
		return nil, fmt.Errorf("failed to create teacher application: %w", err)
	}

	return newApp, nil
}
