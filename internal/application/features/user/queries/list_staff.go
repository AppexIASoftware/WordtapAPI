package queries

import (
	"context"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type StaffUserDTO struct {
	ID         string              `json:"id"`
	Email      string              `json:"email"`
	Name       string              `json:"name"`
	AvatarURL  *string             `json:"avatar_url"`
	Role       entities.UserRole   `json:"role"`
	AccessTier entities.AccessTier `json:"access_tier"`
	IsActive   bool                `json:"is_active"`
	CreatedAt  time.Time           `json:"created_at"`
}

type ListStaffHandler struct {
	staffRepo repositories.StaffRepository
}

func NewListStaffHandler(staffRepo repositories.StaffRepository) *ListStaffHandler {
	return &ListStaffHandler{staffRepo: staffRepo}
}

func (h *ListStaffHandler) Handle(ctx context.Context) ([]StaffUserDTO, error) {
	users, err := h.staffRepo.ListStaff(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]StaffUserDTO, 0, len(users))
	for i := range users {
		u := &users[i]
		result = append(result, ToStaffUserDTO(u))
	}
	return result, nil
}

func ToStaffUserDTO(u *entities.User) StaffUserDTO {
	return StaffUserDTO{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		AvatarURL:  u.AvatarURL,
		Role:       u.Role,
		AccessTier: u.AccessTier,
		IsActive:   u.IsActive,
		CreatedAt:  u.CreatedAt,
	}
}
