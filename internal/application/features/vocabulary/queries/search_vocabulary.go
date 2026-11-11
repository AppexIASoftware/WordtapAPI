package queries

import (
	"context"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type SearchVocabularyQuery struct {
	Term   string
	Limit  int
	Offset int
}

type VocabularySearchResultItem struct {
	ID          string  `json:"id"`
	Word        string  `json:"word"`
	Translation string  `json:"translation"`
	IPA         *string `json:"ipa,omitempty"`
	CEFRLevel   string  `json:"cefr_level"`
	Difficulty  int     `json:"difficulty"`
	Explanation *string `json:"explanation,omitempty"`
	AudioURL    *string `json:"audio_url,omitempty"`
}

type VocabularySearchResult struct {
	Items []VocabularySearchResultItem `json:"items"`
	Total int64                        `json:"total"`
}

type SearchVocabularyHandler struct {
	repo repositories.ContentBankRepository
}

func NewSearchVocabularyHandler(repo repositories.ContentBankRepository) *SearchVocabularyHandler {
	return &SearchVocabularyHandler{repo: repo}
}

func (h *SearchVocabularyHandler) Handle(ctx context.Context, q SearchVocabularyQuery) (*VocabularySearchResult, error) {
	term := strings.TrimSpace(q.Term)
	if term == "" {
		return &VocabularySearchResult{
			Items: make([]VocabularySearchResultItem, 0),
			Total: 0,
		}, nil
	}

	if len(term) > 100 {
		return nil, apperrors.New("El término de búsqueda no puede superar los 100 caracteres.")
	}

	limit := q.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	published := entities.ContentStatusPublished
	filter := repositories.ContentBankFilter{
		Search:    &term,
		CreatedBy: nil,
		OnlyMine:  false,
		Status:    &published,
		Limit:     limit,
		Offset:    offset,
	}

	items, total, err := h.repo.ListItems(ctx, filter)
	if err != nil {
		return nil, err
	}

	res := &VocabularySearchResult{
		Items: make([]VocabularySearchResultItem, 0, len(items)),
		Total: total,
	}

	for _, item := range items {
		res.Items = append(res.Items, VocabularySearchResultItem{
			ID:          item.ID,
			Word:        item.TargetText,
			Translation: item.PromptText,
			IPA:         item.IPA,
			CEFRLevel:   item.CEFRLevel,
			Difficulty:  item.Difficulty,
			Explanation: item.Explanation,
			AudioURL:    item.AudioURL,
		})
	}

	return res, nil
}
