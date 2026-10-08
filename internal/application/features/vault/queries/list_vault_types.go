package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListVaultTypesHandler struct {
	repo repositories.ContentBankRepository
}

func NewListVaultTypesHandler(repo repositories.ContentBankRepository) *ListVaultTypesHandler {
	return &ListVaultTypesHandler{repo: repo}
}

func (h *ListVaultTypesHandler) Handle(ctx context.Context) ([]entities.ContentBankType, error) {
	return h.repo.ListTypes(ctx)
}
