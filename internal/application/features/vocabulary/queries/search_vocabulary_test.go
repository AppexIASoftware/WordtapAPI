package queries_test

import (
	"context"
	"strings"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vocabulary/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type mockContentBankRepository struct {
	lastFilter repositories.ContentBankFilter
	itemsToReturn []entities.ContentBankItem
	totalToReturn int64
	errToReturn   error
}

func (m *mockContentBankRepository) ListTypes(ctx context.Context) ([]entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockContentBankRepository) FindTypeBySlug(ctx context.Context, slug string) (*entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockContentBankRepository) ListCategories(ctx context.Context) ([]entities.ContentCategory, error) {
	return nil, nil
}
func (m *mockContentBankRepository) CreateCategory(ctx context.Context, category *entities.ContentCategory) error {
	return nil
}
func (m *mockContentBankRepository) ListItems(ctx context.Context, filter repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	m.lastFilter = filter
	return m.itemsToReturn, m.totalToReturn, m.errToReturn
}
func (m *mockContentBankRepository) FindItemByID(ctx context.Context, id string) (*entities.ContentBankItem, error) {
	return nil, nil
}
func (m *mockContentBankRepository) CreateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return nil
}
func (m *mockContentBankRepository) UpdateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return nil
}
func (m *mockContentBankRepository) DeleteItem(ctx context.Context, id string) error {
	return nil
}

func TestSearchVocabulary_EmptyTerm(t *testing.T) {
	repo := &mockContentBankRepository{}
	handler := queries.NewSearchVocabularyHandler(repo)

	res, err := handler.Handle(context.Background(), queries.SearchVocabularyQuery{
		Term: "   ",
	})
	if err != nil {
		t.Fatalf("expected no error for empty search, got: %v", err)
	}
	if len(res.Items) != 0 || res.Total != 0 {
		t.Fatalf("expected 0 items and 0 total, got items=%d, total=%d", len(res.Items), res.Total)
	}
}

func TestSearchVocabulary_TermTooLong(t *testing.T) {
	repo := &mockContentBankRepository{}
	handler := queries.NewSearchVocabularyHandler(repo)

	longTerm := strings.Repeat("a", 101)
	_, err := handler.Handle(context.Background(), queries.SearchVocabularyQuery{
		Term: longTerm,
	})
	if err == nil {
		t.Fatal("expected error for term > 100 chars, got nil")
	}
}

func TestSearchVocabulary_ClampsLimitsAndForcesOfficialFilter(t *testing.T) {
	ipa := "/həˈloʊ/"
	expl := "Standard greeting"
	repo := &mockContentBankRepository{
		itemsToReturn: []entities.ContentBankItem{
			{
				ID:          "item-1",
				PromptText:  "hola",
				TargetText:  "hello",
				IPA:         &ipa,
				CEFRLevel:   "A1",
				Difficulty:  1,
				Explanation: &expl,
			},
		},
		totalToReturn: 1,
	}
	handler := queries.NewSearchVocabularyHandler(repo)

	res, err := handler.Handle(context.Background(), queries.SearchVocabularyQuery{
		Term:   "hello",
		Limit:  100, // exceeds max 50
		Offset: -5,  // negative offset
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter.Limit != 20 {
		t.Errorf("expected limit clamped to 20, got %d", repo.lastFilter.Limit)
	}
	if repo.lastFilter.Offset != 0 {
		t.Errorf("expected offset clamped to 0, got %d", repo.lastFilter.Offset)
	}
	if repo.lastFilter.CreatedBy != nil {
		t.Errorf("expected CreatedBy=nil for official vault isolation, got %v", repo.lastFilter.CreatedBy)
	}
	if repo.lastFilter.OnlyMine {
		t.Errorf("expected OnlyMine=false, got true")
	}
	if repo.lastFilter.Status == nil || *repo.lastFilter.Status != entities.ContentStatusPublished {
		t.Errorf("expected Status=published, got %v", repo.lastFilter.Status)
	}

	if res.Total != 1 || len(res.Items) != 1 {
		t.Fatalf("expected 1 item, got total=%d, items=%d", res.Total, len(res.Items))
	}
	item := res.Items[0]
	if item.Word != "hello" || item.Translation != "hola" || *item.IPA != ipa {
		t.Errorf("unexpected mapped fields: %+v", item)
	}
}
