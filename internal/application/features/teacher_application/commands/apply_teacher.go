package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
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
		return nil, apperrors.New("Tu cuenta ya tiene permisos de docente o administrador.")
	}

	// Verificar si ya tiene postulación previa
	existing, err := h.appRepo.FindByUserID(ctx, cmd.UserID)
	if err == nil && existing != nil {
		if existing.Status == entities.ApplicationStatusPending {
			updated := false
			if cmd.Specialty != nil {
				spec := strings.TrimSpace(*cmd.Specialty)
				if spec == "" {
					return nil, apperrors.New("Debes agregar al menos una especialidad académica.")
				}
				if len([]rune(spec)) > 120 {
					return nil, apperrors.New("La especialidad académica no puede superar los 120 caracteres.")
				}
				existing.Specialty = &spec
				updated = true
			}
			if cmd.Bio != nil {
				bio := strings.TrimSpace(*cmd.Bio)
				if bio == defaultAutoApplicationBio || len([]rune(bio)) < 20 {
					return nil, apperrors.New("La biografía académica debe tener al menos 20 caracteres personalizados.")
				}
				if len([]rune(bio)) > 500 {
					return nil, apperrors.New("La biografía académica no puede superar los 500 caracteres.")
				}
				existing.Bio = &bio
				updated = true
			}
			if updated {
				existing.UpdatedAt = time.Now()
				if err := h.appRepo.Update(ctx, existing); err != nil {
					return nil, fmt.Errorf("failed to update pending teacher application: %w", err)
				}
			}
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
