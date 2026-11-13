package commands

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrInvalidStaffEmail = errors.New("valid email is required")
	ErrInvalidStaffRole  = errors.New("role must be admin, moderator, or instructor")
)

type ProvisionStaffCommand struct {
	RequesterID string
	Email       string
	Name        string
	Role        entities.UserRole
}

type ProvisionStaffResult struct {
	User    queries.StaffUserDTO `json:"user"`
	Created bool                 `json:"created"`
}

type ProvisionStaffHandler struct {
	userRepo repositories.StaffRepository
}

func NewProvisionStaffHandler(userRepo repositories.StaffRepository) *ProvisionStaffHandler {
	return &ProvisionStaffHandler{userRepo: userRepo}
}

func (h *ProvisionStaffHandler) Handle(ctx context.Context, cmd ProvisionStaffCommand) (*ProvisionStaffResult, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if email == "" {
		return nil, ErrInvalidStaffEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidStaffEmail
	}

	targetTier, ok := resolveStaffTier(cmd.Role)
	if !ok {
		return nil, ErrInvalidStaffRole
	}

	cleanName := strings.TrimSpace(cmd.Name)
	now := time.Now()

	existing, err := h.userRepo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		roleChanged := existing.Role != cmd.Role
		existing.Role = cmd.Role
		existing.AccessTier = targetTier
		existing.IsActive = true
		if existing.EmailVerifiedAt == nil {
			existing.EmailVerifiedAt = &now
		}
		if cleanName != "" && (existing.Name == "" || existing.Name == strings.Split(email, "@")[0]) {
			existing.Name = cleanName
		}

		if err := h.userRepo.UpdateUser(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update staff user: %w", err)
		}
		if roleChanged {
			_ = h.userRepo.RevokeAllSessions(ctx, existing.ID)
		}

		return &ProvisionStaffResult{
			User:    queries.ToStaffUserDTO(existing),
			Created: false,
		}, nil
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to lookup user by email: %w", err)
	}

	if cleanName == "" {
		cleanName = strings.Split(email, "@")[0]
	}

	newUser := &entities.User{
		Email:             email,
		Name:              cleanName,
		Role:              cmd.Role,
		AccessTier:        targetTier,
		PreferredLanguage: "es",
		LearningLevel:     "advanced",
		Timezone:          "UTC",
		IsActive:          true,
		EmailVerifiedAt:   &now,
	}

	if err := h.userRepo.CreateWithInitialState(ctx, newUser, nil); err != nil {
		return nil, fmt.Errorf("failed to create staff user: %w", err)
	}

	return &ProvisionStaffResult{
		User:    queries.ToStaffUserDTO(newUser),
		Created: true,
	}, nil
}

func resolveStaffTier(role entities.UserRole) (entities.AccessTier, bool) {
	switch role {
	case entities.RoleAdmin:
		return entities.AccessTierAdmin, true
	case entities.RoleModerator, entities.RoleInstructor:
		return entities.AccessTierCourse, true
	default:
		return "", false
	}
}
