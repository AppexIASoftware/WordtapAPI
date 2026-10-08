package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListVaultItemsQuery struct {
	BankType    *string
	CEFRLevel   *string
	Difficulty  *int
	Search      *string
	OnlyMine    bool
	TeacherID   string
	RequesterID string
	Role        entities.UserRole
	Limit       int
	Offset      int
}

type ListVaultItemsResult struct {
	Items []entities.ContentBankItem `json:"items"`
	Total int64                      `json:"total"`
}

type ListVaultItemsHandler struct {
	repo repositories.ContentBankRepository
}

func NewListVaultItemsHandler(repo repositories.ContentBankRepository) *ListVaultItemsHandler {
	return &ListVaultItemsHandler{repo: repo}
}

func (h *ListVaultItemsHandler) Handle(ctx context.Context, q ListVaultItemsQuery) (*ListVaultItemsResult, error) {
	filter := repositories.ContentBankFilter{
		BankType:   q.BankType,
		CEFRLevel:  q.CEFRLevel,
		Difficulty: q.Difficulty,
		Search:     q.Search,
		Limit:      q.Limit,
		Offset:     q.Offset,
	}

	if q.OnlyMine {
		// Biblioteca privada del docente
		teacherID := q.TeacherID
		if teacherID == "" {
			teacherID = q.RequesterID
		}
		filter.CreatedBy = &teacherID
		filter.OnlyMine = true
	} else {
		// Banco oficial Wordtap
		filter.CreatedBy = nil
		filter.OnlyMine = false
		published := entities.ContentStatusPublished
		filter.Status = &published
	}

	items, total, err := h.repo.ListItems(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListVaultItemsResult{
		Items: items,
		Total: total,
	}, nil
}
