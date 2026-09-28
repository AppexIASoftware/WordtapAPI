package rest_test

import (
	"os"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest"
)

func TestNewServer_MissingEnvVariables(t *testing.T) {
	// Limpiar variables de entorno
	os.Unsetenv("CORS_ALLOWED_ORIGINS")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("GOOGLE_CLIENT_ID")

	_, err := rest.NewServer(nil)
	if err == nil {
		t.Fatal("expected error when CORS_ALLOWED_ORIGINS is missing, got nil")
	}

	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	_, err = rest.NewServer(nil)
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is missing, got nil")
	}

	os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-chars-long")
	_, err = rest.NewServer(nil)
	if err == nil {
		t.Fatal("expected error when GOOGLE_CLIENT_ID is missing, got nil")
	}

	os.Setenv("GOOGLE_CLIENT_ID", "test-client-id.apps.googleusercontent.com")
	server, err := rest.NewServer(nil)
	if err != nil {
		t.Fatalf("expected successful server initialization, got error: %v", err)
	}
	if server == nil || server.App == nil {
		t.Fatal("expected valid server instance")
	}
}
