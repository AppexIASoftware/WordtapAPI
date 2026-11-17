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

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	settingCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/commands"
	settingQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
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

	jwtSvc := security.NewJWTService("test-secret-32-bytes-minimum-key!!", 15*time.Minute, time.Hour)
	adminToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{ID: "adm-1", Email: "admin@wordtap.app", Role: entities.RoleAdmin})
	modToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{ID: "mod-1", Email: "mod@wordtap.app", Role: entities.RoleModerator})
	teacherToken, _, _ := jwtSvc.GenerateAccessToken(&entities.User{ID: "tch-1", Email: "teacher@wordtap.app", Role: entities.RoleInstructor})

	getPub := settingQuery.NewGetPublicSettingsHandler(repo)
	getAll := settingQuery.NewGetAllSettingsHandler(repo)
	setCmd := settingCmd.NewSetPlatformSettingHandler(repo)
	eval := access.NewRBACPolicyEvaluator(repo)

	r := setting.NewSettingRouter(getPub, getAll, setCmd).WithRBACEvaluator(eval)
	e := echo.New()
	v1 := e.Group("/api/v1")
	authRequired := restMiddleware.RequireAuth(jwtSvc)
	adminOnly := restMiddleware.RequireRole(entities.RoleAdmin)
	r.RegisterRoutes(v1, authRequired, adminOnly)

	t.Run("GET /api/v1/public-settings works without auth", func(t *testing.T) {
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
	})

	t.Run("admin updates allowed settings and valid RBAC matrix", func(t *testing.T) {
		updateBody, _ := json.Marshal(map[string]string{
			"key":   "contact_email",
			"value": "ayuda@wordtap.org",
		})
		reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(updateBody))
		reqPut.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		reqPut.Header.Set("Authorization", "Bearer "+adminToken)
		recPut := httptest.NewRecorder()
		e.ServeHTTP(recPut, reqPut)
		if recPut.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recPut.Code, recPut.Body.String())
		}
		if repo.settings["contact_email"] != "ayuda@wordtap.org" {
			t.Fatalf("expected ayuda@wordtap.org in repo, got %s", repo.settings["contact_email"])
		}

		rbacBody, _ := json.Marshal(map[string]string{
			"key":   access.RBACSettingKey,
			"value": `{"students:view_directory":{"moderator":true}}`,
		})
		reqRBAC := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rbacBody))
		reqRBAC.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		reqRBAC.Header.Set("Authorization", "Bearer "+adminToken)
		recRBAC := httptest.NewRecorder()
		e.ServeHTTP(recRBAC, reqRBAC)
		if recRBAC.Code != http.StatusOK {
			t.Fatalf("expected 200 for admin RBAC update, got %d: %s", recRBAC.Code, recRBAC.Body.String())
		}

		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
		reqGet.Header.Set("Authorization", "Bearer "+adminToken)
		recGet := httptest.NewRecorder()
		e.ServeHTTP(recGet, reqGet)
		if recGet.Code != http.StatusOK {
			t.Fatalf("expected 200 for admin GET /admin/settings, got %d", recGet.Code)
		}
	})

	t.Run("moderator and instructor are rejected from GET and PUT /admin/settings (403)", func(t *testing.T) {
		prevRBAC := repo.settings[access.RBACSettingKey]
		rbacEscalationBody, _ := json.Marshal(map[string]string{
			"key":   access.RBACSettingKey,
			"value": `{"students:manage_status":{"moderator":true}}`,
		})
		for _, tc := range []struct {
			role  string
			token string
		}{
			{"moderator", modToken},
			{"instructor", teacherToken},
		} {
			reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
			reqGet.Header.Set("Authorization", "Bearer "+tc.token)
			recGet := httptest.NewRecorder()
			e.ServeHTTP(recGet, reqGet)
			if recGet.Code != http.StatusForbidden {
				t.Fatalf("%s GET /admin/settings: expected 403, got %d", tc.role, recGet.Code)
			}

			reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rbacEscalationBody))
			reqPut.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			reqPut.Header.Set("Authorization", "Bearer "+tc.token)
			recPut := httptest.NewRecorder()
			e.ServeHTTP(recPut, reqPut)
			if recPut.Code != http.StatusForbidden {
				t.Fatalf("%s PUT /admin/settings: expected 403, got %d", tc.role, recPut.Code)
			}
		}
		if repo.settings[access.RBACSettingKey] != prevRBAC {
			t.Fatalf("rbac_matrix_policies must remain unchanged after rejected attempts")
		}
	})

	t.Run("moderator still accesses GET /api/v1/rbac/me (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/me", nil)
		req.Header.Set("Authorization", "Bearer "+modToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on /rbac/me for moderator, got %d", rec.Code)
		}
	})

	t.Run("rejects unknown setting keys and malformed RBAC JSON (400)", func(t *testing.T) {
		unknownBody, _ := json.Marshal(map[string]string{
			"key":   "unknown_key",
			"value": "anything",
		})
		reqUnknown := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(unknownBody))
		reqUnknown.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		reqUnknown.Header.Set("Authorization", "Bearer "+adminToken)
		recUnknown := httptest.NewRecorder()
		e.ServeHTTP(recUnknown, reqUnknown)
		if recUnknown.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for unknown_key, got %d: %s", recUnknown.Code, recUnknown.Body.String())
		}

		badJSONBody, _ := json.Marshal(map[string]string{
			"key":   access.RBACSettingKey,
			"value": "{invalid",
		})
		reqBadJSON := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(badJSONBody))
		reqBadJSON.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		reqBadJSON.Header.Set("Authorization", "Bearer "+adminToken)
		recBadJSON := httptest.NewRecorder()
		e.ServeHTTP(recBadJSON, reqBadJSON)
		if recBadJSON.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for malformed rbac json, got %d: %s", recBadJSON.Code, recBadJSON.Body.String())
		}
	})
}
