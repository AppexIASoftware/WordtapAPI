package device_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/device"
)

type mockDeviceRepo struct {
	lastDevice *entities.UserDevice
	revokedUser string
	revokedToken string
}

func (m *mockDeviceRepo) RegisterDevice(ctx context.Context, d *entities.UserDevice) error {
	m.lastDevice = d
	return nil
}

func (m *mockDeviceRepo) RevokeToken(ctx context.Context, userID, token string) error {
	m.revokedUser = userID
	m.revokedToken = token
	return nil
}

func (m *mockDeviceRepo) GetTokensByUser(ctx context.Context, userID string) ([]string, error) {
	return []string{"fcm-token-1"}, nil
}

func (m *mockDeviceRepo) GetTokensByUsers(ctx context.Context, userIDs []string) ([]string, error) {
	return []string{"fcm-token-1"}, nil
}

func (m *mockDeviceRepo) GetAllTokens(ctx context.Context) ([]string, error) {
	return []string{"fcm-token-1"}, nil
}

func (m *mockDeviceRepo) CountActiveTokens(ctx context.Context) (int64, error) {
	return 1, nil
}

func TestDeviceRouter_Register(t *testing.T) {
	repo := &mockDeviceRepo{}
	router := device.NewDeviceRouter(repo)

	e := echo.New()
	v1 := e.Group("/api/v1")
	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "user-456")
			return next(c)
		}
	}
	router.RegisterRoutes(v1, authMiddleware)

	payload := device.RegisterDeviceRequest{
		PushToken: "test-fcm-token-12345",
		Platform:  "android",
	}
	b, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if repo.lastDevice == nil {
		t.Fatal("expected device to be registered")
	}
	if repo.lastDevice.UserID != "user-456" {
		t.Errorf("expected user-456, got %s", repo.lastDevice.UserID)
	}
	if *repo.lastDevice.PushToken != "test-fcm-token-12345" {
		t.Errorf("expected test-fcm-token-12345, got %s", *repo.lastDevice.PushToken)
	}
}

func TestDeviceRouter_Revoke(t *testing.T) {
	repo := &mockDeviceRepo{}
	router := device.NewDeviceRouter(repo)

	e := echo.New()
	v1 := e.Group("/api/v1")
	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "user-456")
			return next(c)
		}
	}
	router.RegisterRoutes(v1, authMiddleware)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/devices/token?token=token-to-revoke", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.revokedUser != "user-456" || repo.revokedToken != "token-to-revoke" {
		t.Errorf("expected revoked token for user-456, got user=%s, token=%s", repo.revokedUser, repo.revokedToken)
	}
}
