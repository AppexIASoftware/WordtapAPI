package commands

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var categorySlugRegex = regexp.MustCompile(`[^a-z0-9]+`)

type CreateVaultCategoryCommand struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type CreateVaultCategoryHandler struct {
	repo repositories.ContentBankRepository
}

func NewCreateVaultCategoryHandler(repo repositories.ContentBankRepository) *CreateVaultCategoryHandler {
	return &CreateVaultCategoryHandler{repo: repo}
}

func (h *CreateVaultCategoryHandler) Handle(ctx context.Context, cmd CreateVaultCategoryCommand) (*entities.ContentCategory, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, apperrors.New("El nombre de la categoría es obligatorio.")
	}

	lower := strings.ToLower(name)
	slug := categorySlugRegex.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "categoria"
	}

	category := &entities.ContentCategory{
		ID:          uuid.NewString(),
		Name:        name,
		Slug:        slug,
		Description: cmd.Description,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	if err := h.repo.CreateCategory(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}
