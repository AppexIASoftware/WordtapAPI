package setting_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	settingCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/commands"
	settingQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/setting"
)

type mockSettingRepo struct {
	settings map[string]string
}

func (m *mockSettingRepo) Get(ctx context.Context, key string) (*entities.PlatformSetting, error) {
	if val, ok := m.settings[key]; ok {
		return &entities.PlatformSetting{Key: key, Value: val, UpdatedAt: time.Now()}, nil
	}
	return nil, nil
}

func (m *mockSettingRepo) Set(ctx context.Context, key string, value string) error {
	m.settings[key] = value
	return nil
}

func (m *mockSettingRepo) GetAll(ctx context.Context) ([]entities.PlatformSetting, error) {
	var list []entities.PlatformSetting
	for k, v := range m.settings {
		list = append(list, entities.PlatformSetting{Key: k, Value: v, UpdatedAt: time.Now()})
	}
	return list, nil
}

func TestSettingRouter(t *testing.T) {
	repo := &mockSettingRepo{settings: map[string]string{
		"contact_email": "soporte@wordtap.app",
		"company_name":  "Wordtap Inc.",
	}}

	getPub := settingQuery.NewGetPublicSettingsHandler(repo)
	getAll := settingQuery.NewGetAllSettingsHandler(repo)
	setCmd := settingCmd.NewSetPlatformSettingHandler(repo)

	r := setting.NewSettingRouter(getPub, getAll, setCmd)
	e := echo.New()
	v1 := e.Group("/api/v1")
	fakeAuth := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	r.RegisterRoutes(v1, fakeAuth, fakeAuth)

	// 1. GET /api/v1/public-settings
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public-settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var pub settingQuery.PublicSettingsDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatalf("failed unmarshaling response: %v", err)
	}
	if pub.ContactEmail != "soporte@wordtap.app" {
		t.Fatalf("expected soporte@wordtap.app, got %s", pub.ContactEmail)
	}

	// 2. PUT /api/v1/admin/settings
	updateBody, _ := json.Marshal(map[string]string{
		"key":   "contact_email",
		"value": "ayuda@wordtap.org",
	})
	reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(updateBody))
	reqPut.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recPut := httptest.NewRecorder()
	e.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recPut.Code)
	}

	if repo.settings["contact_email"] != "ayuda@wordtap.org" {
		t.Fatalf("expected ayuda@wordtap.org in repo, got %s", repo.settings["contact_email"])
	}
}
