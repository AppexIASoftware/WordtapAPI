package security_test

import (
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
)

func TestJWTService_GenerateAndValidate(t *testing.T) {
	jwtSvc := security.NewJWTService("test-secret-key-32-bytes-minimum-size!!", time.Minute, time.Hour)

	user := &entities.User{
		ID:         "test-user-uuid",
		Email:      "student@wordtap.app",
		AccessTier: entities.AccessTierFree,
	}

	token, expiresIn, err := jwtSvc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	if token == "" || expiresIn <= 0 {
		t.Fatalf("invalid token or expiresIn: token=%s, exp=%d", token, expiresIn)
	}

	claims, err := jwtSvc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	if claims.Subject != user.ID {
		t.Errorf("expected subject %s, got %s", user.ID, claims.Subject)
	}
	if claims.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, claims.Email)
	}
	if claims.AccessTier != entities.AccessTierFree {
		t.Errorf("expected tier %s, got %s", entities.AccessTierFree, claims.AccessTier)
	}

	// Test invalid token
	_, err = jwtSvc.ValidateAccessToken("invalid.jwt.token")
	if err == nil {
		t.Error("expected error validating invalid token, got nil")
	}
}

func TestJWTService_RefreshTokenHash(t *testing.T) {
	jwtSvc := security.NewJWTService("secret", time.Minute, time.Hour)

	token1, hash1, exp, err := jwtSvc.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token1) != 64 { // 32 bytes hex encoded = 64 chars
		t.Errorf("expected 64 chars token, got %d", len(token1))
	}
	if hash1 != security.HashToken(token1) {
		t.Errorf("hash mismatch")
	}
	if exp.Before(time.Now()) {
		t.Errorf("refresh token should not be expired")
	}
}
