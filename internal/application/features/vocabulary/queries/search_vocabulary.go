package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/translation"
)

const (
	LangPairEnToEs = "en|es"
	LangPairEsToEn = "es|en"
	maxTermBytes   = 500
)

// TranslationProvider abstracts external translation services (e.g. MyMemory).
type TranslationProvider interface {
	Translate(ctx context.Context, term, langPair string) ([]translation.Entry, error)
}

type SearchVocabularyQuery struct {
	Term     string
	LangPair string
	Limit    int
	Offset   int
}

type VocabularySearchResultItem struct {
	ID          string  `json:"id"`
	Word        string  `json:"word"`
	Translation string  `json:"translation"`
	LangPair    string  `json:"lang_pair"`
	Quality     int     `json:"quality,omitempty"`
	Explanation *string `json:"explanation,omitempty"`
}

type VocabularySearchResult struct {
	Items []VocabularySearchResultItem `json:"items"`
	Total int64                        `json:"total"`
}

type SearchVocabularyHandler struct {
	translator TranslationProvider
}

func NewSearchVocabularyHandler(translator TranslationProvider) *SearchVocabularyHandler {
	return &SearchVocabularyHandler{translator: translator}
}

func (h *SearchVocabularyHandler) Handle(ctx context.Context, q SearchVocabularyQuery) (*VocabularySearchResult, error) {
	term := strings.TrimSpace(q.Term)
	if term == "" {
		return &VocabularySearchResult{
			Items: make([]VocabularySearchResultItem, 0),
			Total: 0,
		}, nil
	}

	if len([]byte(term)) > maxTermBytes {
		return nil, apperrors.New("El término de búsqueda no puede superar los 500 bytes.")
	}

	langPair := strings.TrimSpace(q.LangPair)
	if langPair == "" {
		langPair = LangPairEsToEn
	}
	if langPair != LangPairEnToEs && langPair != LangPairEsToEn {
		return nil, apperrors.New("El par de idiomas debe ser 'en|es' o 'es|en'.")
	}

	limit := q.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	entries, err := h.translator.Translate(ctx, term, langPair)
	if err != nil {
		return nil, err
	}

	total := int64(len(entries))
	if offset >= len(entries) {
		return &VocabularySearchResult{
			Items: make([]VocabularySearchResultItem, 0),
			Total: total,
		}, nil
	}

	end := offset + limit
	if end > len(entries) {
		end = len(entries)
	}
	page := entries[offset:end]

	res := &VocabularySearchResult{
		Items: make([]VocabularySearchResultItem, 0, len(page)),
		Total: total,
	}

	for _, entry := range page {
		var explanation *string
		if entry.Quality > 0 {
			note := fmt.Sprintf("Precisión de traducción: %d%%", entry.Quality)
			explanation = &note
		}
		res.Items = append(res.Items, VocabularySearchResultItem{
			ID:          entry.ID,
			Word:        entry.Source,
			Translation: entry.Translation,
			LangPair:    langPair,
			Quality:     entry.Quality,
			Explanation: explanation,
		})
	}

	return res, nil
}
