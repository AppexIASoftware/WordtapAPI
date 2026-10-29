package vault_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	vaultCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	vaultQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vault"
)

type mockContentBankRepo struct {
	types      []entities.ContentBankType
	categories []entities.ContentCategory
	items      map[string]*entities.ContentBankItem
}

func newMockContentBankRepo() *mockContentBankRepo {
	return &mockContentBankRepo{
		types: []entities.ContentBankType{
			{Slug: "vocabulary", DisplayName: "Vocabulario", IsActive: true},
			{Slug: "phrase", DisplayName: "Oraciones", IsActive: true},
		},
		categories: []entities.ContentCategory{
			{Name: "Viajes & Turismo", Slug: "viajes-turismo", IsActive: true},
			{Name: "Gastronomía & Restaurantes", Slug: "gastronomia-restaurantes", IsActive: true},
		},
		items: make(map[string]*entities.ContentBankItem),
	}
}

func (m *mockContentBankRepo) ListTypes(ctx context.Context) ([]entities.ContentBankType, error) {
	return m.types, nil
}

func (m *mockContentBankRepo) ListCategories(ctx context.Context) ([]entities.ContentCategory, error) {
	return m.categories, nil
}

func (m *mockContentBankRepo) CreateCategory(ctx context.Context, category *entities.ContentCategory) error {
	m.categories = append(m.categories, *category)
	return nil
}

func (m *mockContentBankRepo) FindTypeBySlug(ctx context.Context, slug string) (*entities.ContentBankType, error) {
	for _, t := range m.types {
		if t.Slug == slug {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockContentBankRepo) ListItems(ctx context.Context, filter repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	var list []entities.ContentBankItem
	for _, item := range m.items {
		if filter.OnlyMine && filter.CreatedBy != nil {
			if item.CreatedBy == nil || *item.CreatedBy != *filter.CreatedBy {
				continue
			}
		} else if !filter.OnlyMine && filter.CreatedBy == nil {
			if item.CreatedBy != nil {
				continue
			}
		}
		list = append(list, *item)
	}
	return list, int64(len(list)), nil
}

func (m *mockContentBankRepo) FindItemByID(ctx context.Context, id string) (*entities.ContentBankItem, error) {
	if it, ok := m.items[id]; ok {
		return it, nil
	}
	return nil, nil
}

func (m *mockContentBankRepo) CreateItem(ctx context.Context, item *entities.ContentBankItem) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockContentBankRepo) UpdateItem(ctx context.Context, item *entities.ContentBankItem) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockContentBankRepo) DeleteItem(ctx context.Context, id string) error {
	delete(m.items, id)
	return nil
}

func TestVaultRouter_OfficialAndPrivateFlow(t *testing.T) {
	repo := newMockContentBankRepo()
	listTypes := vaultQuery.NewListVaultTypesHandler(repo)
	listCategories := vaultQuery.NewListVaultCategoriesHandler(repo)
	createCategory := vaultCmd.NewCreateVaultCategoryHandler(repo)
	listItems := vaultQuery.NewListVaultItemsHandler(repo)
	createItem := vaultCmd.NewCreateVaultItemHandler(repo)
	updateItem := vaultCmd.NewUpdateVaultItemHandler(repo)
	deleteItem := vaultCmd.NewDeleteVaultItemHandler(repo)

	router := vault.NewVaultRouter(listTypes, listCategories, createCategory, listItems, createItem, updateItem, deleteItem)

	e := echo.New()
	v1 := e.Group("/api/v1")

	// Middleware que inyecta contexto según header para tests
	mockAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			uid := c.Request().Header.Get("X-Test-User-ID")
			if uid == "" {
				uid = "user-123"
			}
			role := c.Request().Header.Get("X-Test-Role")
			if role == "" {
				role = "admin"
			}
			c.Set("user_id", uid)
			c.Set("user_role", role)
			return next(c)
		}
	}
	passThrough := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error { return next(c) }
	}

	router.RegisterRoutes(v1, mockAuth, passThrough, passThrough, passThrough)

	// 1. List types
	t.Run("GET /vault/types", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/vault/types", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	// 1.1 List categories
	t.Run("GET /vault/categories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/vault/categories", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	// 1.2 Teacher creates category
	t.Run("POST /vault/categories (Teacher)", func(t *testing.T) {
		desc := "Vocabulario de finanzas"
		body := vault.CreateVaultCategoryRequest{
			Name:        "Finanzas & Negocios",
			Description: &desc,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vault/categories", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", "teacher-1")
		req.Header.Set("X-Test-Role", "instructor")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var cat entities.ContentCategory
		_ = json.Unmarshal(rec.Body.Bytes(), &cat)
		if cat.Name != "Finanzas & Negocios" {
			t.Errorf("expected category name 'Finanzas & Negocios', got '%s'", cat.Name)
		}
	})

	// 2. Admin creates official vault item
	var officialID string
	t.Run("POST /vault/items (Admin)", func(t *testing.T) {
		body := vault.CreateVaultItemRequest{
			BankType:   "vocabulary",
			PromptText: "Hola",
			TargetText: "Hello",
			CEFRLevel:  "A1",
			Difficulty: 1,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vault/items", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", "admin-1")
		req.Header.Set("X-Test-Role", "admin")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var item entities.ContentBankItem
		_ = json.Unmarshal(rec.Body.Bytes(), &item)
		if item.CreatedBy != nil {
			t.Errorf("expected CreatedBy to be nil for official item, got %v", item.CreatedBy)
		}
		officialID = item.ID
	})

	// 3. Teacher creates private vault item
	var privateID string
	t.Run("POST /teacher/vault/items (Teacher)", func(t *testing.T) {
		body := vault.CreateVaultItemRequest{
			BankType:   "phrase",
			PromptText: "¿Cómo estás?",
			TargetText: "How are you doing?",
			CEFRLevel:  "A2",
			Difficulty: 1,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/vault/items", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", "teacher-99")
		req.Header.Set("X-Test-Role", "instructor")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var item entities.ContentBankItem
		_ = json.Unmarshal(rec.Body.Bytes(), &item)
		if item.CreatedBy == nil || *item.CreatedBy != "teacher-99" {
			t.Errorf("expected CreatedBy to be teacher-99, got %v", item.CreatedBy)
		}
		privateID = item.ID
	})

	// 4. Teacher lists official items (should see official, NOT private items)
	t.Run("GET /vault/items (Teacher reading official catalog)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/vault/items", nil)
		req.Header.Set("X-Test-User-ID", "teacher-99")
		req.Header.Set("X-Test-Role", "instructor")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var res vaultQuery.ListVaultItemsResult
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		if res.Total != 1 || res.Items[0].ID != officialID {
			t.Errorf("expected 1 official item, got %d", res.Total)
		}
	})

	// 5. Teacher lists private library
	t.Run("GET /teacher/vault/items (Teacher reading own library)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/vault/items", nil)
		req.Header.Set("X-Test-User-ID", "teacher-99")
		req.Header.Set("X-Test-Role", "instructor")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var res vaultQuery.ListVaultItemsResult
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		if res.Total != 1 || res.Items[0].ID != privateID {
			t.Errorf("expected 1 private item, got %d", res.Total)
		}
	})

	// 6. Update private item
	t.Run("PUT /teacher/vault/items/:id (Teacher updates own item)", func(t *testing.T) {
		newPrompt := "¿Cómo estás hoy?"
		newType := "collocation"
		updateBody := vault.UpdateVaultItemRequest{
			BankType:   &newType,
			PromptText: &newPrompt,
		}
		raw, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/teacher/vault/items/"+privateID, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", "teacher-99")
		req.Header.Set("X-Test-Role", "instructor")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var item entities.ContentBankItem
		_ = json.Unmarshal(rec.Body.Bytes(), &item)
		if item.PromptText != newPrompt {
			t.Errorf("expected prompt to be updated to %s, got %s", newPrompt, item.PromptText)
		}
		if item.BankType != newType {
			t.Errorf("expected bank_type to be updated to %s, got %s", newType, item.BankType)
		}
	})

	// 7. Delete private item
	t.Run("DELETE /teacher/vault/items/:id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/teacher/vault/items/"+privateID, nil)
		req.Header.Set("X-Test-User-ID", "teacher-99")
		req.Header.Set("X-Test-Role", "instructor")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if _, ok := repo.items[privateID]; ok {
			t.Errorf("expected private item to be deleted from repo")
		}
	})
}
