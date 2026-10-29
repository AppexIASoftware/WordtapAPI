package commands

import (
	"context"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type UpdateVaultItemCommand struct {
	ID          string
	BankType    *string
	CategoryID  *string
	CategoryIDSet bool
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
		return nil, apperrors.New("No se encontró el ítem.")
	}

	if item.CreatedBy == nil {
		// Ítem Oficial
		if cmd.Role != entities.RoleAdmin && cmd.Role != entities.RoleModerator {
			return nil, apperrors.New("Solo administradores y moderadores pueden actualizar ítems oficiales.")
		}
	} else {
		// Ítem Privado de Docente
		if cmd.Role != entities.RoleAdmin && *item.CreatedBy != cmd.RequesterID {
			return nil, apperrors.New("Solo puedes actualizar tus propios ítems.")
		}
	}

	if cmd.BankType != nil {
		trimmed := strings.TrimSpace(*cmd.BankType)
		if trimmed != "" {
			item.BankType = trimmed
		}
	}
	if cmd.CategoryIDSet || cmd.CategoryID != nil {
		item.CategoryID = cmd.CategoryID
		item.Category = nil
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
