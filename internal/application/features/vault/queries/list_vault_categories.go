package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ListVaultCategoriesHandler procesa la consulta de categorías semánticas del Content Vault.
type ListVaultCategoriesHandler struct {
	repo repositories.ContentBankRepository
}

// NewListVaultCategoriesHandler construye el query handler desacoplado.
func NewListVaultCategoriesHandler(repo repositories.ContentBankRepository) *ListVaultCategoriesHandler {
	return &ListVaultCategoriesHandler{repo: repo}
}

// Handle retorna las categorías de contenido activas en la plataforma.
func (h *ListVaultCategoriesHandler) Handle(ctx context.Context) ([]entities.ContentCategory, error) {
	return h.repo.ListCategories(ctx)
}
