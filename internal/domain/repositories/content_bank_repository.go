package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// ContentBankFilter encapsula los criterios de búsqueda sobre el banco de contenidos.
type ContentBankFilter struct {
	BankType   *string
	CEFRLevel  *string
	Difficulty *int
	Search     *string
	CreatedBy  *string // nil para ítems oficiales (WHERE created_by IS NULL), o UUID del docente
	OnlyMine   bool    // true para filtrar estrictamente created_by = teacherID
	Status     *entities.ContentStatus
	Limit      int
	Offset     int
}

// ContentBankRepository define el contrato de persistencia para el Content Vault.
type ContentBankRepository interface {
	ListTypes(ctx context.Context) ([]entities.ContentBankType, error)
	FindTypeBySlug(ctx context.Context, slug string) (*entities.ContentBankType, error)
	ListItems(ctx context.Context, filter ContentBankFilter) ([]entities.ContentBankItem, int64, error)
	FindItemByID(ctx context.Context, id string) (*entities.ContentBankItem, error)
	CreateItem(ctx context.Context, item *entities.ContentBankItem) error
	UpdateItem(ctx context.Context, item *entities.ContentBankItem) error
	DeleteItem(ctx context.Context, id string) error
}
