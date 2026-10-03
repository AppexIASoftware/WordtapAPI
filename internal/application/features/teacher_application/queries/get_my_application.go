package queries

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type GetMyApplicationQuery struct {
	UserID string
}

type TeacherApplicationStatusDTO struct {
	Status          string  `json:"status"` // "none", "pending", "approved", "rejected"
	ApplicationID   *string `json:"application_id,omitempty"`
	Bio             *string `json:"bio,omitempty"`
	Specialty       *string `json:"specialty,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
}

type GetMyApplicationHandler struct {
	appRepo repositories.TeacherApplicationRepository
}

func NewGetMyApplicationHandler(appRepo repositories.TeacherApplicationRepository) *GetMyApplicationHandler {
	return &GetMyApplicationHandler{appRepo: appRepo}
}

func (h *GetMyApplicationHandler) Handle(ctx context.Context, q GetMyApplicationQuery) (*TeacherApplicationStatusDTO, error) {
	if q.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	app, err := h.appRepo.FindByUserID(ctx, q.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &TeacherApplicationStatusDTO{Status: "none"}, nil
		}
		return nil, fmt.Errorf("failed fetching application: %w", err)
	}

	createdStr := app.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	return &TeacherApplicationStatusDTO{
		Status:          string(app.Status),
		ApplicationID:   &app.ID,
		Bio:             app.Bio,
		Specialty:       app.Specialty,
		RejectionReason: app.RejectionReason,
		CreatedAt:       &createdStr,
	}, nil
}
