package vocabulary_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vocabulary/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vocabulary"
)

type mockContentBankRepo struct {
	items []entities.ContentBankItem
	total int64
}

func (m *mockContentBankRepo) ListTypes(ctx context.Context) ([]entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockContentBankRepo) FindTypeBySlug(ctx context.Context, slug string) (*entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockContentBankRepo) ListCategories(ctx context.Context) ([]entities.ContentCategory, error) {
	return nil, nil
}
func (m *mockContentBankRepo) CreateCategory(ctx context.Context, category *entities.ContentCategory) error {
	return nil
}
func (m *mockContentBankRepo) ListItems(ctx context.Context, filter repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	return m.items, m.total, nil
}
func (m *mockContentBankRepo) FindItemByID(ctx context.Context, id string) (*entities.ContentBankItem, error) {
	return nil, nil
}
func (m *mockContentBankRepo) CreateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return nil
}
func (m *mockContentBankRepo) UpdateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return nil
}
func (m *mockContentBankRepo) DeleteItem(ctx context.Context, id string) error {
	return nil
}

func TestVocabularyRouter_Search(t *testing.T) {
	ipa := "/kæt/"
	repo := &mockContentBankRepo{
		items: []entities.ContentBankItem{
			{
				ID:         "v-1",
				PromptText: "gato",
				TargetText: "cat",
				IPA:        &ipa,
				CEFRLevel:  "A1",
				Difficulty: 1,
			},
		},
		total: 1,
	}

	handler := queries.NewSearchVocabularyHandler(repo)
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vocabulary/search?q=cat", nil)
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
