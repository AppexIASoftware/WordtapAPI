package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/auth"
)

type fakeUserRepo struct {
	mu           sync.Mutex
	users        map[string]*entities.User
	sessions     map[string]*entities.UserSession
	authAccounts map[string]*entities.UserAuthAccount
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:        map[string]*entities.User{},
		sessions:     map[string]*entities.UserSession{},
		authAccounts: map[string]*entities.UserAuthAccount{},
	}
}

func authAccountKey(provider, providerAccountID string) string {
	return provider + ":" + providerAccountID
}

func (r *fakeUserRepo) FindByID(_ context.Context, id string) (*entities.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*entities.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		if strings.EqualFold(user.Email, email) {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) FindAuthAccount(_ context.Context, provider, providerAccountID string) (*entities.UserAuthAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.authAccounts[authAccountKey(provider, providerAccountID)]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	account.User = r.users[account.UserID]
	return account, nil
}

func (r *fakeUserRepo) CreateWithInitialState(_ context.Context, user *entities.User, _ *entities.UserAuthAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user.ID == "" {
		user.ID = "user-" + user.Email
	}
	r.users[user.ID] = user
	return nil
}

func (r *fakeUserRepo) CreateAuthAccount(_ context.Context, authAccount *entities.UserAuthAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authAccounts[authAccountKey(authAccount.Provider, authAccount.ProviderAccountID)] = authAccount
	return nil
}

func (r *fakeUserRepo) CreateSession(_ context.Context, session *entities.UserSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session.ID == "" {
		session.ID = "session-" + session.SessionTokenHash[:8]
	}
	r.sessions[session.SessionTokenHash] = session
	return nil
}

func (r *fakeUserRepo) FindSessionByTokenHash(_ context.Context, tokenHash string) (*entities.UserSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[tokenHash]
	if !ok || session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
		return nil, gorm.ErrRecordNotFound
	}
	found := *session
	if found.User == nil {
		found.User = r.users[found.UserID]
	}
	return &found, nil
}

// RotateSession mirrors the SQL guard: it revokes and creates atomically, and
// reports false when another caller already consumed the token.
func (r *fakeUserRepo) RotateSession(_ context.Context, oldTokenHash string, newSession *entities.UserSession) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[oldTokenHash]
	now := time.Now()
	if !ok || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return false, nil
	}

	session.RevokedAt = &now
	if newSession.ID == "" {
		newSession.ID = "session-" + newSession.SessionTokenHash[:8]
	}
	r.sessions[newSession.SessionTokenHash] = newSession
	return true, nil
}

func (r *fakeUserRepo) RevokeSession(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session, ok := r.sessions[tokenHash]; ok {
		now := time.Now()
		session.RevokedAt = &now
	}
	return nil
}

func (r *fakeUserRepo) RevokeAllSessions(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, session := range r.sessions {
		if session.UserID == userID {
			session.RevokedAt = &now
		}
	}
	return nil
}

func (r *fakeUserRepo) UpdateUser(context.Context, *entities.User) error {
	return nil
}

func newTestServer(t *testing.T, repo *fakeUserRepo, devLoginEnabled bool) *echo.Echo {
	t.Helper()

	jwtService := security.NewJWTService("test-secret-key-32-bytes-minimum-size!!", time.Minute, time.Hour)
	router := auth.NewAuthRouter(nil, repo, nil, jwtService, devLoginEnabled)

	e := echo.New()
	v1 := e.Group("/api/v1")
	noopAuth := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, noopAuth)
	return e
}

type fakeGoogleVerifier struct {
	claims *security.GoogleTokenClaims
	err    error
}

func (f *fakeGoogleVerifier) VerifyIDToken(context.Context, string) (*security.GoogleTokenClaims, error) {
	return f.claims, f.err
}

func newTestServerWithLogin(t *testing.T, repo *fakeUserRepo, verifier security.GoogleVerifier) *echo.Echo {
	t.Helper()

	jwtService := security.NewJWTService("test-secret-key-32-bytes-minimum-size!!", time.Minute, time.Hour)
	handler := commands.NewLoginWithGoogleHandler(repo, verifier, jwtService, nil)
	router := auth.NewAuthRouter(handler, repo, nil, jwtService, false)

	e := echo.New()
	v1 := e.Group("/api/v1")
	noopAuth := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, noopAuth)
	return e
}

func seedUser(repo *fakeUserRepo) *entities.User {
	user := &entities.User{
		ID:                "user-1",
		Email:             "student@wordtap.app",
		Name:              "Student",
		Role:              entities.RoleStudent,
		AccessTier:        entities.AccessTierFree,
		PreferredLanguage: "es",
		LearningLevel:     "beginner",
		Timezone:          "UTC",
		IsActive:          true,
	}
	repo.users[user.ID] = user
	return user
}

func seedSession(repo *fakeUserRepo, user *entities.User, token string) {
	hash := security.HashToken(token)
	repo.sessions[hash] = &entities.UserSession{
		ID:               "session-1",
		UserID:           user.ID,
		SessionTokenHash: hash,
		ExpiresAt:        time.Now().Add(time.Hour),
		User:             user,
	}
}

func postJSON(e *echo.Echo, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRefreshRotatesToken(t *testing.T) {
	repo := newFakeUserRepo()
	user := seedUser(repo)
	seedSession(repo, user, "old-refresh-token")

	e := newTestServer(t, repo, false)
	rec := postJSON(e, "/api/v1/auth/refresh", `{"refresh_token":"old-refresh-token"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result commands.AuthResultDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("expected a new access and refresh token pair")
	}

	oldSession := repo.sessions[security.HashToken("old-refresh-token")]
	if oldSession.RevokedAt == nil {
		t.Fatal("expected the presented refresh token to be revoked")
	}
	if _, ok := repo.sessions[security.HashToken(result.RefreshToken)]; !ok {
		t.Fatal("expected the rotated refresh token to be persisted")
	}
}

func TestRefreshRejectsUnknownToken(t *testing.T) {
	repo := newFakeUserRepo()
	e := newTestServer(t, repo, false)

	rec := postJSON(e, "/api/v1/auth/refresh", `{"refresh_token":"does-not-exist"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRefreshRejectsRevokedToken(t *testing.T) {
	repo := newFakeUserRepo()
	user := seedUser(repo)
	seedSession(repo, user, "revoked-token")
	now := time.Now()
	repo.sessions[security.HashToken("revoked-token")].RevokedAt = &now

	e := newTestServer(t, repo, false)
	rec := postJSON(e, "/api/v1/auth/refresh", `{"refresh_token":"revoked-token"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRefreshIsAtomicUnderConcurrentRequests(t *testing.T) {
	repo := newFakeUserRepo()
	user := seedUser(repo)
	seedSession(repo, user, "concurrent-refresh-token")

	e := newTestServer(t, repo, false)

	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rec := postJSON(e, "/api/v1/auth/refresh", `{"refresh_token":"concurrent-refresh-token"}`)
			codes[idx] = rec.Code
		}(i)
	}
	wg.Wait()

	okCount, unauthorizedCount := 0, 0
	for _, code := range codes {
		switch code {
		case http.StatusOK:
			okCount++
		case http.StatusUnauthorized:
			unauthorizedCount++
		default:
			t.Fatalf("unexpected status code %d", code)
		}
	}
	if okCount != 1 || unauthorizedCount != 1 {
		t.Fatalf("expected exactly one 200 and one 401, got %d and %d", okCount, unauthorizedCount)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	activeSessions := 0
	for _, session := range repo.sessions {
		if session.RevokedAt == nil {
			activeSessions++
		}
	}
	if activeSessions != 1 {
		t.Fatalf("expected exactly one active session, got %d", activeSessions)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	repo := newFakeUserRepo()
	user := seedUser(repo)
	seedSession(repo, user, "session-to-close")

	e := newTestServer(t, repo, false)
	rec := postJSON(e, "/api/v1/auth/logout", `{"refresh_token":"session-to-close"}`)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if repo.sessions[security.HashToken("session-to-close")].RevokedAt == nil {
		t.Fatal("expected the session to be revoked")
	}
}

func TestDevLoginDisabledReturnsNotFound(t *testing.T) {
	repo := newFakeUserRepo()
	e := newTestServer(t, repo, false)

	rec := postJSON(e, "/api/v1/auth/dev-login", `{}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDevLoginEnabledIssuesSession(t *testing.T) {
	repo := newFakeUserRepo()
	repo.users["dev-user"] = &entities.User{
		ID:                "dev-user",
		Email:             "mateo.silva@wordtap.app",
		Name:              "Mateo",
		Role:              entities.RoleInstructor,
		AccessTier:        entities.AccessTierCourse,
		PreferredLanguage: "es",
		LearningLevel:     "advanced",
		Timezone:          "UTC",
		IsActive:          true,
	}

	e := newTestServer(t, repo, true)
	rec := postJSON(e, "/api/v1/auth/dev-login", `{}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result commands.AuthResultDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.RefreshToken == "" {
		t.Fatal("expected a refresh token")
	}
	if _, ok := repo.sessions[security.HashToken(result.RefreshToken)]; !ok {
		t.Fatal("expected the dev session to be persisted")
	}
}
