package commands

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
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
		return apperrors.New("No se encontró el ítem.")
	}

	if item.CreatedBy == nil {
		if cmd.Role != entities.RoleAdmin && cmd.Role != entities.RoleModerator {
			return apperrors.New("Solo administradores y moderadores pueden eliminar ítems oficiales.")
		}
	} else {
		if cmd.Role != entities.RoleAdmin && *item.CreatedBy != cmd.RequesterID {
			return apperrors.New("Solo puedes eliminar tus propios ítems.")
		}
	}

	return h.repo.DeleteItem(ctx, cmd.ID)
}
