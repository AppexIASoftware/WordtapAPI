package rest_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest"
)

func TestInboxRequiresBearerToken(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("JWT_SECRET", "test-secret-key-at-least-32-chars-long")
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")

	server, err := rest.NewServer(nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, route := range []struct{ method, path string }{
		{"GET", "/api/v1/notifications"},
		{"GET", "/api/v1/notifications/unread-count"},
		{"PUT", "/api/v1/notifications/b2a3a528-4591-4bee-bfbb-88e06965d87c/read"},
	} {
		rec := httptest.NewRecorder()
		server.App.ServeHTTP(rec, httptest.NewRequest(route.method, route.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status %d", route.method, route.path, rec.Code)
		}
	}
}

func TestNotificationPreferencesRequireAuthentication(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("JWT_SECRET", "test-secret-key-at-least-32-chars-long")
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")

	server, err := rest.NewServer(nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			server.App.ServeHTTP(rec, httptest.NewRequest(method, "/api/v1/notification-preferences", nil))
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d", rec.Code)
			}
		})
	}
}

func TestServerSecurityMiddleware(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("JWT_SECRET", "test-secret-key-at-least-32-chars-long")
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")

	server, err := rest.NewServer(nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("rejects payloads above 2 MiB with 413", func(t *testing.T) {
		oversized := bytes.Repeat([]byte("a"), 2*1024*1024+128)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", bytes.NewReader(oversized))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "198.51.100.55:1111"
		rec := httptest.NewRecorder()
		server.App.ServeHTTP(rec, req)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413 Request Entity Too Large, got %d", rec.Code)
		}
	})

	t.Run("throttles excessive /api/v1/auth burst with 429", func(t *testing.T) {
		var throttled bool
		for i := 0; i < 20; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{}`))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "198.51.100.99:2222"
			rec := httptest.NewRecorder()
			server.App.ServeHTTP(rec, req)
			if rec.Code == http.StatusTooManyRequests {
				throttled = true
				break
			}
		}
		if !throttled {
			t.Fatal("expected /api/v1/auth burst above 15 requests to trigger 429")
		}
	})
}
