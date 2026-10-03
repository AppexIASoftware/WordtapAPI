package teacher_application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	appCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/teacher_application/commands"
	appQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/teacher_application/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/teacher_application"
)

type mockUserRepo struct{}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*entities.User, error) {
	return &entities.User{ID: id, Role: entities.RoleStudent}, nil
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepo) CreateWithInitialState(ctx context.Context, u *entities.User, a *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepo) UpdateUser(ctx context.Context, u *entities.User) error { return nil }
func (m *mockUserRepo) FindAuthAccount(ctx context.Context, p, a string) (*entities.UserAuthAccount, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepo) CreateAuthAccount(ctx context.Context, a *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepo) CreateSession(ctx context.Context, s *entities.UserSession) error { return nil }
func (m *mockUserRepo) FindSessionByTokenHash(ctx context.Context, t string) (*entities.UserSession, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepo) RevokeSession(ctx context.Context, t string) error { return nil }
func (m *mockUserRepo) RevokeAllSessions(ctx context.Context, u string) error { return nil }

type mockAppRepo struct {
	apps []entities.TeacherApplication
}

func (m *mockAppRepo) Create(ctx context.Context, app *entities.TeacherApplication) error {
	m.apps = append(m.apps, *app)
	return nil
}
func (m *mockAppRepo) FindByID(ctx context.Context, id string) (*entities.TeacherApplication, error) {
	for _, a := range m.apps {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockAppRepo) FindByUserID(ctx context.Context, userID string) (*entities.TeacherApplication, error) {
	for _, a := range m.apps {
		if a.UserID == userID {
			return &a, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockAppRepo) List(ctx context.Context, s *entities.ApplicationStatus, l, o int) ([]entities.TeacherApplication, int64, error) {
	return m.apps, int64(len(m.apps)), nil
}
func (m *mockAppRepo) Update(ctx context.Context, app *entities.TeacherApplication) error { return nil }
func (m *mockAppRepo) ApproveInTx(ctx context.Context, appID string, revID string) error {
	for i := range m.apps {
		if m.apps[i].ID == appID {
			m.apps[i].Status = entities.ApplicationStatusApproved
			return nil
		}
	}
	return nil
}
func (m *mockAppRepo) RejectInTx(ctx context.Context, appID string, revID string, reason string) error {
	for i := range m.apps {
		if m.apps[i].ID == appID {
			m.apps[i].Status = entities.ApplicationStatusRejected
			return nil
		}
	}
	return nil
}
func (m *mockAppRepo) SuspendInTx(ctx context.Context, appID string, revID string, reason string) error {
	for i := range m.apps {
		if m.apps[i].ID == appID {
			m.apps[i].Status = entities.ApplicationStatusSuspended
			return nil
		}
	}
	return nil
}
func (m *mockAppRepo) ReactivateInTx(ctx context.Context, appID string, revID string) error {
	for i := range m.apps {
		if m.apps[i].ID == appID {
			m.apps[i].Status = entities.ApplicationStatusApproved
			return nil
		}
	}
	return nil
}
func (m *mockAppRepo) DeleteInTx(ctx context.Context, appID string) error {
	filtered := make([]entities.TeacherApplication, 0)
	for _, a := range m.apps {
		if a.ID != appID {
			filtered = append(filtered, a)
		}
	}
	m.apps = filtered
	return nil
}

func setupTestRouter() (*echo.Echo, *mockAppRepo) {
	e := echo.New()
	uRepo := &mockUserRepo{}
	aRepo := &mockAppRepo{}

	applyH := appCmd.NewApplyTeacherHandler(aRepo, uRepo)
	revH := appCmd.NewReviewApplicationHandler(aRepo)
	delH := appCmd.NewDeleteApplicationHandler(aRepo)
	myStatusH := appQuery.NewGetMyApplicationHandler(aRepo)
	listH := appQuery.NewListApplicationsHandler(aRepo)

	router := teacher_application.NewTeacherApplicationRouter(applyH, revH, delH, myStatusH, listH)

	fakeAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "test-user-id")
			c.Set("user_role", "admin")
			return next(c)
		}
	}

	v1 := e.Group("/api/v1")
	router.RegisterRoutes(v1, fakeAuth, fakeAuth)
	return e, aRepo
}

func TestApplyAndGetStatusRoutes(t *testing.T) {
	e, _ := setupTestRouter()

	// 1. Postularse
	body, _ := json.Marshal(map[string]string{"specialty": "English"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher-applications", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rec.Code)
	}

	// 2. Consultar propio estado
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/teacher-applications/my-status", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec2.Code)
	}
}

func TestSuspendReactivateDeleteRoutes(t *testing.T) {
	e, repo := setupTestRouter()
	repo.apps = append(repo.apps, entities.TeacherApplication{
		ID:     "app-approved-1",
		UserID: "user-1",
		Status: entities.ApplicationStatusApproved,
	})

	// 1. Suspend
	body := []byte(`{"reason":"Violación de términos pedagógicos"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/teacher-applications/app-approved-1/suspend", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on suspend, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.apps[0].Status != entities.ApplicationStatusSuspended {
		t.Fatalf("expected status suspended, got %s", repo.apps[0].Status)
	}

	// 2. Reactivate
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/teacher-applications/app-approved-1/reactivate", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on reactivate, got %d: %s", rec2.Code, rec2.Body.String())
	}
	if repo.apps[0].Status != entities.ApplicationStatusApproved {
		t.Fatalf("expected status approved after reactivate, got %s", repo.apps[0].Status)
	}

	// 3. Delete
	req3 := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/teacher-applications/app-approved-1", nil)
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete, got %d: %s", rec3.Code, rec3.Body.String())
	}
	if len(repo.apps) != 0 {
		t.Fatalf("expected 0 apps after delete, got %d", len(repo.apps))
	}
}
