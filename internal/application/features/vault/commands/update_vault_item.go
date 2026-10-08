package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type UpdateVaultItemCommand struct {
	ID          string
	PromptText  *string
	TargetText  *string
	IPA         *string
	CEFRLevel   *string
	Difficulty  *int
	AudioURL    *string
	Explanation *string
	Metadata    *string
	RequesterID string
	Role        entities.UserRole
}

type UpdateVaultItemHandler struct {
	repo repositories.ContentBankRepository
}

func NewUpdateVaultItemHandler(repo repositories.ContentBankRepository) *UpdateVaultItemHandler {
	return &UpdateVaultItemHandler{repo: repo}
}

func (h *UpdateVaultItemHandler) Handle(ctx context.Context, cmd UpdateVaultItemCommand) (*entities.ContentBankItem, error) {
	item, err := h.repo.FindItemByID(ctx, cmd.ID)
	if err != nil {
		return nil, errors.New("vault item not found")
	}

	if item.CreatedBy == nil {
		// Ítem Oficial
		if cmd.Role != entities.RoleAdmin && cmd.Role != entities.RoleModerator {
			return nil, errors.New("forbidden: only administrators and moderators can update official vault items")
		}
	} else {
		// Ítem Privado de Docente
		if cmd.Role != entities.RoleAdmin && *item.CreatedBy != cmd.RequesterID {
			return nil, errors.New("forbidden: you can only update your own private vault items")
		}
	}

	if cmd.PromptText != nil {
		trimmed := strings.TrimSpace(*cmd.PromptText)
		if trimmed != "" {
			item.PromptText = trimmed
		}
	}
	if cmd.TargetText != nil {
		trimmed := strings.TrimSpace(*cmd.TargetText)
		if trimmed != "" {
			item.TargetText = trimmed
		}
	}
	if cmd.IPA != nil {
		item.IPA = cmd.IPA
	}
	if cmd.CEFRLevel != nil {
		item.CEFRLevel = strings.ToUpper(strings.TrimSpace(*cmd.CEFRLevel))
	}
	if cmd.Difficulty != nil && *cmd.Difficulty >= 1 && *cmd.Difficulty <= 3 {
		item.Difficulty = *cmd.Difficulty
	}
	if cmd.AudioURL != nil {
		item.AudioURL = cmd.AudioURL
	}
	if cmd.Explanation != nil {
		item.Explanation = cmd.Explanation
	}
	if cmd.Metadata != nil {
		item.Metadata = *cmd.Metadata
	}
	item.UpdatedAt = time.Now()

	if err := h.repo.UpdateItem(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}
