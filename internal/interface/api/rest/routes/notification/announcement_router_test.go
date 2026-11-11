package notification_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationdispatch"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	notificationRoute "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/notification"
)

type mockUserRepoForRoute struct {
	userIDs []string
}

func (m *mockUserRepoForRoute) FindByID(context.Context, string) (*entities.User, error) { return nil, nil }
func (m *mockUserRepoForRoute) FindByEmail(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepoForRoute) FindAuthAccount(context.Context, string, string) (*entities.UserAuthAccount, error) {
	return nil, nil
}
func (m *mockUserRepoForRoute) CreateWithInitialState(context.Context, *entities.User, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepoForRoute) CreateAuthAccount(context.Context, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepoForRoute) CreateSession(context.Context, *entities.UserSession) error { return nil }
func (m *mockUserRepoForRoute) FindSessionByTokenHash(context.Context, string) (*entities.UserSession, error) {
	return nil, nil
}
func (m *mockUserRepoForRoute) RotateSession(context.Context, string, *entities.UserSession) (bool, error) {
	return false, nil
}
func (m *mockUserRepoForRoute) RevokeSession(context.Context, string) error     { return nil }
func (m *mockUserRepoForRoute) RevokeAllSessions(context.Context, string) error { return nil }
func (m *mockUserRepoForRoute) UpdateUser(context.Context, *entities.User) error {
	return nil
}
func (m *mockUserRepoForRoute) ListUserIDs(context.Context, *entities.UserRole, *string) ([]string, error) {
	return m.userIDs, nil
}

type mockInboxRepoForRoute struct{}

func (m *mockInboxRepoForRoute) List(context.Context, string, int, *inbox.Cursor) (inbox.Page, error) {
	return inbox.Page{}, nil
}
func (m *mockInboxRepoForRoute) UnreadCount(context.Context, string) (int64, error) { return 0, nil }
func (m *mockInboxRepoForRoute) MarkRead(context.Context, string, string) (bool, error) {
	return true, nil
}
func (m *mockInboxRepoForRoute) Create(context.Context, *entities.Notification) error { return nil }
func (m *mockInboxRepoForRoute) CreateBatch(context.Context, []entities.Notification) error {
	return nil
}

type mockDeviceRepoForRoute struct {
	tokens []string
}

func (m *mockDeviceRepoForRoute) RegisterDevice(context.Context, *entities.UserDevice) error {
	return nil
}
func (m *mockDeviceRepoForRoute) RevokeToken(context.Context, string, string) error { return nil }
func (m *mockDeviceRepoForRoute) GetTokensByUser(context.Context, string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForRoute) GetTokensByUsers(context.Context, []string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForRoute) GetAllTokens(context.Context) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForRoute) CountActiveTokens(context.Context) (int64, error) {
	return int64(len(m.tokens)), nil
}

func TestAnnouncementRouter_BroadcastAndStats(t *testing.T) {
	userRepo := &mockUserRepoForRoute{userIDs: []string{"usr-1", "usr-2"}}
	inboxRepo := &mockInboxRepoForRoute{}
	deviceRepo := &mockDeviceRepoForRoute{tokens: []string{"tok-1"}}

	handler := notificationdispatch.NewBroadcastAnnouncementHandler(userRepo, inboxRepo, deviceRepo, nil)
	router := notificationRoute.NewAnnouncementRouter(handler, deviceRepo)

	e := echo.New()
	v1 := e.Group("/api/v1")
	passMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, passMiddleware, passMiddleware)

	t.Run("POST /api/v1/admin/announcements éxito", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"title": "Mantenimiento",
			"body":  "Programado para las 22:00",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /api/v1/admin/notifications/stats éxito", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/notifications/stats", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if count, ok := resp["active_devices_count"].(float64); !ok || int(count) != 1 {
			t.Errorf("esperado active_devices_count = 1, obtenido %v", resp["active_devices_count"])
		}
	})
}
