package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	userCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	userQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
)

type routerUserRepoStub struct {
	users map[string]*entities.User
}

func (s *routerUserRepoStub) FindByID(_ context.Context, id string) (*entities.User, error) {
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (s *routerUserRepoStub) FindByEmail(_ context.Context, email string) (*entities.User, error) {
	if u, ok := s.users[email]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (s *routerUserRepoStub) FindAuthAccount(context.Context, string, string) (*entities.UserAuthAccount, error) {
	return nil, gorm.ErrRecordNotFound
}
func (s *routerUserRepoStub) CreateWithInitialState(_ context.Context, user *entities.User, _ *entities.UserAuthAccount) error {
	user.ID = "new-admin-id"
	user.CreatedAt = time.Now()
	s.users[user.Email] = user
	return nil
}
func (s *routerUserRepoStub) CreateAuthAccount(context.Context, *entities.UserAuthAccount) error { return nil }
func (s *routerUserRepoStub) CreateSession(context.Context, *entities.UserSession) error         { return nil }
func (s *routerUserRepoStub) FindSessionByTokenHash(context.Context, string) (*entities.UserSession, error) {
	return nil, gorm.ErrRecordNotFound
}
func (s *routerUserRepoStub) RotateSession(context.Context, string, *entities.UserSession) (bool, error) {
	return false, nil
}
func (s *routerUserRepoStub) RevokeSession(context.Context, string) error    { return nil }
func (s *routerUserRepoStub) RevokeAllSessions(context.Context, string) error { return nil }
func (s *routerUserRepoStub) UpdateUser(_ context.Context, user *entities.User) error {
	s.users[user.Email] = user
	return nil
}
func (s *routerUserRepoStub) ListUserIDs(context.Context, *entities.UserRole, *string) ([]string, error) {
	return nil, nil
}
func (s *routerUserRepoStub) ListStaff(context.Context) ([]entities.User, error) {
	out := make([]entities.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, *u)
	}
	return out, nil
}
func (s *routerUserRepoStub) DeleteUser(_ context.Context, userID string) error {
	for email, u := range s.users {
		if u.ID == userID {
			delete(s.users, email)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func setupStaffRouterTest(role entities.UserRole) *echo.Echo {
	e := echo.New()
	v1 := e.Group("/api/v1")
	repo := &routerUserRepoStub{users: map[string]*entities.User{
		"admin-1": {ID: "admin-1", Email: "admin@wordtap.io", Name: "Admin", Role: entities.RoleAdmin, AccessTier: entities.AccessTierAdmin, IsActive: true},
	}}

	authStub := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if role == "" {
				return c.NoContent(http.StatusUnauthorized)
			}
			c.Set("user_id", "admin-1")
			c.Set("user_role", string(role))
			return next(c)
		}
	}
	adminOrMod := restMiddleware.RequireRole(entities.RoleAdmin, entities.RoleModerator)
	adminOnly := restMiddleware.RequireRole(entities.RoleAdmin)

	router := NewStaffRouter(
		userQuery.NewListStaffHandler(repo),
		userCmd.NewProvisionStaffHandler(repo),
		userCmd.NewUpdateStaffStatusHandler(repo),
		userCmd.NewDeleteStaffHandler(repo),
	)
	router.RegisterRoutes(v1, authStub, adminOrMod, adminOnly)
	return e
}

func TestStaffRouterRBAC(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"email": "nuevo.admin@wordtap.io",
		"name":  "Nuevo Admin",
		"role":  "admin",
	})

	t.Run("moderator cannot provision new admin (403)", func(t *testing.T) {
		e := setupStaffRouterTest(entities.RoleModerator)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/staff", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("admin can provision new admin (201)", func(t *testing.T) {
		e := setupStaffRouterTest(entities.RoleAdmin)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/staff", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("moderator can list staff read-only (200)", func(t *testing.T) {
		e := setupStaffRouterTest(entities.RoleModerator)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/staff", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}
