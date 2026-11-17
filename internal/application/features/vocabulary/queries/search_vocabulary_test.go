package queries_test

import (
	"context"
	"strings"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vocabulary/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/translation"
)

type mockTranslator struct {
	lastTerm     string
	lastLangPair string
	entries      []translation.Entry
	err          error
}

func (m *mockTranslator) Translate(_ context.Context, term, langPair string) ([]translation.Entry, error) {
	m.lastTerm = term
	m.lastLangPair = langPair
	return m.entries, m.err
}

func TestSearchVocabulary_EmptyTerm(t *testing.T) {
	mock := &mockTranslator{}
	handler := queries.NewSearchVocabularyHandler(mock)

	res, err := handler.Handle(context.Background(), queries.SearchVocabularyQuery{Term: "   "})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res.Items) != 0 || res.Total != 0 {
		t.Fatalf("expected empty result, got %+v", res)
	}
}

func TestSearchVocabulary_TermTooLong(t *testing.T) {
	mock := &mockTranslator{}
	handler := queries.NewSearchVocabularyHandler(mock)

	_, err := handler.Handle(context.Background(), queries.SearchVocabularyQuery{
		Term: strings.Repeat("a", 501),
	})
	if err == nil {
		t.Fatal("expected error for term exceeding 500 bytes")
	}
}

func TestSearchVocabulary_InvalidLangPair(t *testing.T) {
	mock := &mockTranslator{}
	handler := queries.NewSearchVocabularyHandler(mock)

	_, err := handler.Handle(context.Background(), queries.SearchVocabularyQuery{
		Term:     "hello",
		LangPair: "fr|de",
	})
	if err == nil {
		t.Fatal("expected error for unsupported langpair")
	}
}

func TestSearchVocabulary_DefaultsToEsEnAndPaginates(t *testing.T) {
	mock := &mockTranslator{
		entries: []translation.Entry{
			{ID: "1", Source: "gato", Translation: "cat", Quality: 100},
			{ID: "2", Source: "gato", Translation: "kitty", Quality: 75},
		},
	}
	handler := queries.NewSearchVocabularyHandler(mock)

	res, err := handler.Handle(context.Background(), queries.SearchVocabularyQuery{
		Term:   "gato",
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastLangPair != queries.LangPairEsToEn {
		t.Fatalf("expected default langpair %s, got %s", queries.LangPairEsToEn, mock.lastLangPair)
	}
	if res.Total != 2 || len(res.Items) != 1 || res.Items[0].Translation != "cat" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Items[0].Explanation == nil || *res.Items[0].Explanation != "Precisión de traducción: 100%" || strings.Contains(*res.Items[0].Explanation, "MyMemory") {
		t.Fatalf("unexpected explanation: %v", res.Items[0].Explanation)
	}
}
