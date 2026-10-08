package commands

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type DeleteVaultItemCommand struct {
	ID          string
	RequesterID string
	Role        entities.UserRole
}

type DeleteVaultItemHandler struct {
	repo repositories.ContentBankRepository
}

func NewDeleteVaultItemHandler(repo repositories.ContentBankRepository) *DeleteVaultItemHandler {
	return &DeleteVaultItemHandler{repo: repo}
}

func (h *DeleteVaultItemHandler) Handle(ctx context.Context, cmd DeleteVaultItemCommand) error {
	item, err := h.repo.FindItemByID(ctx, cmd.ID)
	if err != nil {
		return errors.New("vault item not found")
	}

	if item.CreatedBy == nil {
		if cmd.Role != entities.RoleAdmin && cmd.Role != entities.RoleModerator {
			return errors.New("forbidden: only administrators and moderators can delete official vault items")
		}
	} else {
		if cmd.Role != entities.RoleAdmin && *item.CreatedBy != cmd.RequesterID {
			return errors.New("forbidden: you can only delete your own private vault items")
		}
	}

	return h.repo.DeleteItem(ctx, cmd.ID)
}
