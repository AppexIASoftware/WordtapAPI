package vocabulary_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vocabulary/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/translation"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vocabulary"
)

type mockTranslator struct {
	entries []translation.Entry
}

func (m *mockTranslator) Translate(_ context.Context, _, _ string) ([]translation.Entry, error) {
	return m.entries, nil
}

func TestVocabularyRouter_Search(t *testing.T) {
	translator := &mockTranslator{
		entries: []translation.Entry{
			{
				ID:          "primary",
				Source:      "cat",
				Translation: "gato",
				Quality:     100,
			},
		},
	}

	handler := queries.NewSearchVocabularyHandler(translator)
	router := vocabulary.NewVocabularyRouter(handler)

	e := echo.New()
	v1 := e.Group("/api/v1")
	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "user-123")
			c.Set("user_role", "student")
			return next(c)
		}
	}
	router.RegisterRoutes(v1, authMiddleware)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vocabulary/search?q=cat&langpair=en|es", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res queries.VocabularySearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Total != 1 || len(res.Items) != 1 {
		t.Fatalf("expected 1 result, got total=%d, items=%d", res.Total, len(res.Items))
	}
	if res.Items[0].Word != "cat" || res.Items[0].Translation != "gato" {
		t.Errorf("unexpected item content: %+v", res.Items[0])
	}
}
