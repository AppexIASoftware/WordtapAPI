package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/security"
)

var errUnexpectedRepoCall = errors.New("unexpected repository call")

type fakeGoogleVerifier struct {
	claims *security.GoogleTokenClaims
	err    error
}

func (f *fakeGoogleVerifier) VerifyIDToken(context.Context, string) (*security.GoogleTokenClaims, error) {
	return f.claims, f.err
}

type unusedUserRepo struct{}

func (unusedUserRepo) FindByID(context.Context, string) (*entities.User, error) {
	return nil, errUnexpectedRepoCall
}

func (unusedUserRepo) FindByEmail(context.Context, string) (*entities.User, error) {
	return nil, errUnexpectedRepoCall
}

func (unusedUserRepo) FindAuthAccount(context.Context, string, string) (*entities.UserAuthAccount, error) {
	return nil, errUnexpectedRepoCall
}

func (unusedUserRepo) CreateWithInitialState(context.Context, *entities.User, *entities.UserAuthAccount) error {
	return errUnexpectedRepoCall
}

func (unusedUserRepo) CreateAuthAccount(context.Context, *entities.UserAuthAccount) error {
	return errUnexpectedRepoCall
}

func (unusedUserRepo) CreateSession(context.Context, *entities.UserSession) error {
	return errUnexpectedRepoCall
}

func (unusedUserRepo) FindSessionByTokenHash(context.Context, string) (*entities.UserSession, error) {
	return nil, errUnexpectedRepoCall
}

func (unusedUserRepo) RotateSession(context.Context, string, *entities.UserSession) (bool, error) {
	return false, errUnexpectedRepoCall
}

func (unusedUserRepo) RevokeSession(context.Context, string) error {
	return errUnexpectedRepoCall
}

func (unusedUserRepo) RevokeAllSessions(context.Context, string) error {
	return errUnexpectedRepoCall
}

func (unusedUserRepo) UpdateUser(context.Context, *entities.User) error {
	return errUnexpectedRepoCall
}

func TestLoginWithGoogleRejectsUnverifiedEmail(t *testing.T) {
	verifier := &fakeGoogleVerifier{
		claims: &security.GoogleTokenClaims{
			Sub:           "google-1",
			Email:         "user@example.com",
			EmailVerified: false,
		},
	}
	jwtService := security.NewJWTService("test-secret-key-32-bytes-minimum-size!!", time.Minute, time.Hour)
	handler := commands.NewLoginWithGoogleHandler(unusedUserRepo{}, verifier, jwtService, nil)

	_, err := handler.Handle(context.Background(), commands.LoginWithGoogleCommand{IDToken: "id-token"})
	if !errors.Is(err, commands.ErrGoogleEmailNotVerified) {
		t.Fatalf("expected ErrGoogleEmailNotVerified, got %v", err)
	}
}

// existingUserRepo devuelve un usuario ya registrado por correo, como ocurre
// cuando el mismo correo llega por un proveedor distinto a Google.
type existingUserRepo struct {
	unusedUserRepo
	user           *entities.User
	createdAccount *entities.UserAuthAccount
	createdSession *entities.UserSession
}

func (r *existingUserRepo) FindByEmail(context.Context, string) (*entities.User, error) {
	return r.user, nil
}

func (r *existingUserRepo) CreateAuthAccount(_ context.Context, account *entities.UserAuthAccount) error {
	r.createdAccount = account
	return nil
}

func (r *existingUserRepo) CreateSession(_ context.Context, session *entities.UserSession) error {
	r.createdSession = session
	return nil
}

func TestLoginWithGoogleBlocksLinkingUnverifiedEmail(t *testing.T) {
	verifier := &fakeGoogleVerifier{
		claims: &security.GoogleTokenClaims{
			Sub:           "google-2",
			Email:         "student@wordtap.app",
			EmailVerified: true,
		},
	}
	repo := &existingUserRepo{
		user: &entities.User{
			ID:       "user-existing",
			Email:    "student@wordtap.app",
			Name:     "Student",
			IsActive: true,
		},
	}
	jwtService := security.NewJWTService("test-secret-key-32-bytes-minimum-size!!", time.Minute, time.Hour)
	handler := commands.NewLoginWithGoogleHandler(repo, verifier, jwtService, nil)

	_, err := handler.Handle(context.Background(), commands.LoginWithGoogleCommand{IDToken: "id-token"})
	if !errors.Is(err, commands.ErrAccountLinkRequired) {
		t.Fatalf("expected ErrAccountLinkRequired, got %v", err)
	}
	if repo.createdAccount != nil {
		t.Fatal("expected no Google account to be linked to an unverified user")
	}
}

func TestLoginWithGoogleLinksVerifiedEmail(t *testing.T) {
	verifier := &fakeGoogleVerifier{
		claims: &security.GoogleTokenClaims{
			Sub:           "google-3",
			Email:         "student@wordtap.app",
			EmailVerified: true,
		},
	}
	verifiedAt := time.Now()
	repo := &existingUserRepo{
		user: &entities.User{
			ID:              "user-verified",
			Email:           "student@wordtap.app",
			Name:            "Student",
			IsActive:        true,
			EmailVerifiedAt: &verifiedAt,
		},
	}
	jwtService := security.NewJWTService("test-secret-key-32-bytes-minimum-size!!", time.Minute, time.Hour)
	handler := commands.NewLoginWithGoogleHandler(repo, verifier, jwtService, nil)

	result, err := handler.Handle(context.Background(), commands.LoginWithGoogleCommand{IDToken: "id-token"})
	if err != nil {
		t.Fatalf("expected the verified account to be linked, got %v", err)
	}
	if repo.createdAccount == nil {
		t.Fatal("expected the Google account to be linked to the verified user")
	}
	if repo.createdSession == nil {
		t.Fatal("expected a session to be persisted for the linked user")
	}
	if result.RefreshToken == "" || result.AccessToken == "" {
		t.Fatal("expected a token pair for the linked user")
	}
}
