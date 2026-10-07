package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
)

func TestRequireAuthMiddleware(t *testing.T) {
	jwtSvc := security.NewJWTService("secret-key-for-middleware-testing", time.Minute, time.Hour)
	authMiddleware := middleware.RequireAuth(jwtSvc)

	user := &entities.User{
		ID:         "usr-123",
		Email:      "auth@wordtap.app",
		AccessTier: entities.AccessTierFree,
	}
	validToken, _, _ := jwtSvc.GenerateAccessToken(user)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "missing header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed header",
			authHeader:     "Basic 12345",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.payload",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := authMiddleware(func(ctx *echo.Context) error {
				return ctx.NoContent(http.StatusOK)
			})

			_ = handler(c)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequireRoleMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		userRole       string
		allowedRoles   []entities.UserRole
		expectedStatus int
	}{
		{
			name:           "allowed admin role",
			userRole:       "admin",
			allowedRoles:   []entities.UserRole{entities.RoleAdmin},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "forbidden student accessing admin",
			userRole:       "student",
			allowedRoles:   []entities.UserRole{entities.RoleAdmin, entities.RoleInstructor},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_role", tt.userRole)

			roleMiddleware := middleware.RequireRole(tt.allowedRoles...)
			handler := roleMiddleware(func(ctx *echo.Context) error {
				return ctx.NoContent(http.StatusOK)
			})

			_ = handler(c)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	jwtSvc := security.NewJWTService("secret-key-for-optional-auth", time.Minute, time.Hour)
	optionalMiddleware := middleware.OptionalAuth(jwtSvc)

	user := &entities.User{
		ID:         "usr-opt-1",
		Email:      "opt@wordtap.app",
		Role:       entities.RoleInstructor,
		AccessTier: entities.AccessTierFree,
	}
	validToken, _, _ := jwtSvc.GenerateAccessToken(user)

	tests := []struct {
		name         string
		authHeader   string
		expectUserID string
		expectRole   string
	}{
		{
			name:         "anonymous request without header",
			authHeader:   "",
			expectUserID: "",
			expectRole:   "",
		},
		{
			name:         "invalid header format",
			authHeader:   "Basic 12345",
			expectUserID: "",
			expectRole:   "",
		},
		{
			name:         "invalid or expired token",
			authHeader:   "Bearer bogus.token.value",
			expectUserID: "",
			expectRole:   "",
		},
		{
			name:         "valid token populates context",
			authHeader:   "Bearer " + validToken,
			expectUserID: "usr-opt-1",
			expectRole:   "instructor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/public-or-private", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			var capturedUserID, capturedRole string
			handler := optionalMiddleware(func(ctx *echo.Context) error {
				capturedUserID, _ = ctx.Get("user_id").(string)
				capturedRole, _ = ctx.Get("user_role").(string)
				return ctx.NoContent(http.StatusOK)
			})

			_ = handler(c)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200 OK, got %d", rec.Code)
			}
			if capturedUserID != tt.expectUserID {
				t.Errorf("expected user_id '%s', got '%s'", tt.expectUserID, capturedUserID)
			}
			if capturedRole != tt.expectRole {
				t.Errorf("expected user_role '%s', got '%s'", tt.expectRole, capturedRole)
			}
		})
	}
}
