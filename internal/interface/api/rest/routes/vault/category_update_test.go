package vault_test

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/labstack/echo/v5"
 vaultCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
 "github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
 "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vault"
)

func TestVaultCategoryUpdate(t *testing.T) {
 for _, tc := range []struct{name, body, want string}{
  {"omitted", `{}`, "category-1"},
  {"null", `{"category_id":null}`, ""},
  {"value", `{"category_id":"category-2"}`, "category-2"},
 } {
  t.Run(tc.name, func(t *testing.T) {
   repo := newMockContentBankRepo()
   category := "category-1"
   repo.items["item-1"] = &entities.ContentBankItem{ID:"item-1", CategoryID:&category}
   router := vault.NewVaultRouter(nil, nil, nil, nil, nil, vaultCmd.NewUpdateVaultItemHandler(repo), nil)
   e := echo.New()
   auth := func(next echo.HandlerFunc) echo.HandlerFunc { return func(c *echo.Context) error { c.Set("user_id", "admin"); c.Set("user_role", "admin"); return next(c) } }
   pass := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
   router.RegisterRoutes(e.Group("/api/v1"), auth, pass, pass, pass)
   req := httptest.NewRequest(http.MethodPut, "/api/v1/vault/items/item-1", strings.NewReader(tc.body))
   req.Header.Set("Content-Type", "application/json")
   rec := httptest.NewRecorder()
   e.ServeHTTP(rec, req)
   if rec.Code != http.StatusOK { t.Fatalf("status %d: %s", rec.Code, rec.Body.String()) }
   got := repo.items["item-1"].CategoryID
   if tc.want == "" { if got != nil { t.Fatalf("expected nil, got %q", *got) }; return }
   if got == nil || *got != tc.want { t.Fatalf("expected %s, got %v", tc.want, got) }
  })
 }
}
