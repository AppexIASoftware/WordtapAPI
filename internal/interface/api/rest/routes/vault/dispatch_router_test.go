package vault_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	vaultCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/vault"
)

type mockBankForDispatchRoute struct {
	item *entities.ContentBankItem
}

func (m *mockBankForDispatchRoute) ListTypes(context.Context) ([]entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockBankForDispatchRoute) FindTypeBySlug(context.Context, string) (*entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockBankForDispatchRoute) ListCategories(context.Context) ([]entities.ContentCategory, error) {
	return nil, nil
}
func (m *mockBankForDispatchRoute) CreateCategory(context.Context, *entities.ContentCategory) error {
	return nil
}
func (m *mockBankForDispatchRoute) ListItems(context.Context, repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	return nil, 0, nil
}
func (m *mockBankForDispatchRoute) FindItemByID(context.Context, string) (*entities.ContentBankItem, error) {
	return m.item, nil
}
func (m *mockBankForDispatchRoute) CreateItem(context.Context, *entities.ContentBankItem) error {
	return nil
}
func (m *mockBankForDispatchRoute) UpdateItem(context.Context, *entities.ContentBankItem) error {
	return nil
}
func (m *mockBankForDispatchRoute) DeleteItem(context.Context, string) error { return nil }

type mockClassroomForDispatchRoute struct {
	classrooms  map[string]*entities.Classroom
	enrollments map[string][]entities.ClassroomEnrollment
}

func (m *mockClassroomForDispatchRoute) Create(context.Context, *entities.Classroom) error {
	return nil
}
func (m *mockClassroomForDispatchRoute) FindByID(_ context.Context, id string) (*entities.Classroom, error) {
	return m.classrooms[id], nil
}
func (m *mockClassroomForDispatchRoute) FindByCode(context.Context, string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomForDispatchRoute) FindByTeacher(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomForDispatchRoute) FindByStudent(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomForDispatchRoute) EnrollStudent(context.Context, *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockClassroomForDispatchRoute) FindEnrollment(context.Context, string, string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomForDispatchRoute) ListEnrollments(_ context.Context, clsID string) ([]entities.ClassroomEnrollment, error) {
	return m.enrollments[clsID], nil
}
func (m *mockClassroomForDispatchRoute) ListStumbles(context.Context, string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockClassroomForDispatchRoute) CreateStumble(context.Context, *entities.StudentStumble) error {
	return nil
}
func (m *mockClassroomForDispatchRoute) Update(context.Context, *entities.Classroom) error {
	return nil
}

type mockUserForDispatchRoute struct{}

func (m *mockUserForDispatchRoute) FindByID(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserForDispatchRoute) FindByEmail(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserForDispatchRoute) FindAuthAccount(context.Context, string, string) (*entities.UserAuthAccount, error) {
	return nil, nil
}
func (m *mockUserForDispatchRoute) CreateWithInitialState(context.Context, *entities.User, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserForDispatchRoute) CreateAuthAccount(context.Context, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserForDispatchRoute) CreateSession(context.Context, *entities.UserSession) error {
	return nil
}
func (m *mockUserForDispatchRoute) FindSessionByTokenHash(context.Context, string) (*entities.UserSession, error) {
	return nil, nil
}
func (m *mockUserForDispatchRoute) RotateSession(context.Context, string, *entities.UserSession) (bool, error) {
	return false, nil
}
func (m *mockUserForDispatchRoute) RevokeSession(context.Context, string) error { return nil }
func (m *mockUserForDispatchRoute) RevokeAllSessions(context.Context, string) error {
	return nil
}
func (m *mockUserForDispatchRoute) UpdateUser(context.Context, *entities.User) error { return nil }
func (m *mockUserForDispatchRoute) ListUserIDs(context.Context, *entities.UserRole, *string) ([]string, error) {
	return []string{"student-1"}, nil
}

type mockInboxForDispatchRoute struct{}

func (m *mockInboxForDispatchRoute) List(context.Context, string, int, *inbox.Cursor) (inbox.Page, error) {
	return inbox.Page{}, nil
}
func (m *mockInboxForDispatchRoute) UnreadCount(context.Context, string) (int64, error) { return 0, nil }
func (m *mockInboxForDispatchRoute) MarkRead(context.Context, string, string) (bool, error) {
	return true, nil
}
func (m *mockInboxForDispatchRoute) Create(context.Context, *entities.Notification) error { return nil }
func (m *mockInboxForDispatchRoute) CreateBatch(context.Context, []entities.Notification) error {
	return nil
}

type mockDeviceForDispatchRoute struct{}

func (m *mockDeviceForDispatchRoute) RegisterDevice(context.Context, *entities.UserDevice) error {
	return nil
}
func (m *mockDeviceForDispatchRoute) RevokeToken(context.Context, string, string) error { return nil }
func (m *mockDeviceForDispatchRoute) GetTokensByUser(context.Context, string) ([]string, error) {
	return []string{"tok-1"}, nil
}
func (m *mockDeviceForDispatchRoute) GetTokensByUsers(context.Context, []string) ([]string, error) {
	return []string{"tok-1"}, nil
}
func (m *mockDeviceForDispatchRoute) GetAllTokens(context.Context) ([]string, error) {
	return []string{"tok-1"}, nil
}
func (m *mockDeviceForDispatchRoute) CountActiveTokens(context.Context) (int64, error) {
	return 1, nil
}

func TestDispatchRouter_DispatchRBAC(t *testing.T) {
	bankRepo := &mockBankForDispatchRoute{
		item: &entities.ContentBankItem{
			ID:         "item-456",
			PromptText: "casa",
			TargetText: "house",
			CEFRLevel:  "A1",
		},
	}
	clsRepo := &mockClassroomForDispatchRoute{
		classrooms: map[string]*entities.Classroom{
			"cls-own":   {ID: "cls-own", TeacherID: "teacher-1", Name: "Grupo A"},
			"cls-other": {ID: "cls-other", TeacherID: "teacher-2", Name: "Grupo B"},
		},
		enrollments: map[string][]entities.ClassroomEnrollment{
			"cls-own": {{ID: "en-1", ClassroomID: "cls-own", StudentID: "student-1"}},
		},
	}
	userRepo := &mockUserForDispatchRoute{}
	inboxRepo := &mockInboxForDispatchRoute{}
	deviceRepo := &mockDeviceForDispatchRoute{}

	rbacEval := access.NewRBACPolicyEvaluator(nil)
	handler := vaultCmd.NewDispatchVaultWordHandler(bankRepo, clsRepo, userRepo, inboxRepo, deviceRepo, nil).
		WithRBACEvaluator(rbacEval)
	router := vault.NewDispatchRouter(handler)

	e := echo.New()
	v1 := e.Group("/api/v1")
	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", c.Request().Header.Get("X-Test-User-ID"))
			c.Set("user_role", c.Request().Header.Get("X-Test-Role"))
			return next(c)
		}
	}
	dispatchAccess := restMiddleware.RequireAnyPermission(
		rbacEval,
		access.PermNotificationsClassroomBroadcast,
		access.PermNotificationsGlobalAnnouncements,
	)
	router.RegisterRoutes(v1, authMiddleware, dispatchAccess)

	doDispatch := func(userID, role string, payload map[string]any) *httptest.ResponseRecorder {
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vault/items/item-456/dispatch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User-ID", userID)
		req.Header.Set("X-Test-Role", role)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	t.Run("admin dispatches all (200)", func(t *testing.T) {
		rec := doDispatch("admin-1", "admin", map[string]any{"target_audience": "all"})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("moderator dispatches cefr (200)", func(t *testing.T) {
		rec := doDispatch("mod-1", "moderator", map[string]any{"target_audience": "cefr", "target_cefr_level": "A1"})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("instructor is blocked from all and cefr (403)", func(t *testing.T) {
		for _, aud := range []string{"all", "cefr"} {
			rec := doDispatch("teacher-1", "instructor", map[string]any{"target_audience": aud})
			if rec.Code != http.StatusForbidden {
				t.Fatalf("audience %s: expected 403, got %d: %s", aud, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("instructor is blocked from another teacher classroom (403)", func(t *testing.T) {
		rec := doDispatch("teacher-1", "instructor", map[string]any{
			"target_audience":     "classroom",
			"target_classroom_id": "cls-other",
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("instructor dispatches to own classroom (200)", func(t *testing.T) {
		rec := doDispatch("teacher-1", "instructor", map[string]any{
			"target_audience":     "classroom",
			"target_classroom_id": "cls-own",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("student is blocked (403)", func(t *testing.T) {
		rec := doDispatch("student-1", "student", map[string]any{"target_audience": "all"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
