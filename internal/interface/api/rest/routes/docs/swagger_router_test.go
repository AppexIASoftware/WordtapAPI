package docs_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/docs"
)

func TestSwaggerRouter(t *testing.T) {
	e := echo.New()
	router := docs.NewSwaggerRouter()
	router.RegisterRoutes(e)

	t.Run("GET /swagger sirve interfaz HTML de Swagger UI", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperaba status 200, obtuvo %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "SwaggerUIBundle") {
			t.Fatalf("la respuesta no contiene SwaggerUIBundle: %s", body)
		}
		if !strings.Contains(body, "/swagger/doc.json") {
			t.Fatalf("la respuesta no apunta a /swagger/doc.json")
		}
	})

	t.Run("GET /swagger/doc.json entrega especificación OpenAPI válida", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperaba status 200, obtuvo %d", rec.Code)
		}

		var parsed map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
			t.Fatalf("la respuesta no es un JSON válido: %v", err)
		}

		info, ok := parsed["info"].(map[string]any)
		if !ok || info["title"] != "WordtapAPI" {
			t.Fatalf("esperaba info.title == 'WordtapAPI', obtuvo: %v", info)
		}

		paths, ok := parsed["paths"].(map[string]any)
		if !ok || len(paths) == 0 {
			t.Fatalf("esperaba paths poblados en el spec de OpenAPI")
		}
	})
}
