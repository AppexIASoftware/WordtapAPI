package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// UserRepository define el contrato de persistencia para usuarios, cuentas OAuth y sesiones.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entities.User, error)
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	FindAuthAccount(ctx context.Context, provider, providerAccountID string) (*entities.UserAuthAccount, error)
	CreateWithInitialState(ctx context.Context, user *entities.User, authAccount *entities.UserAuthAccount) error
	CreateAuthAccount(ctx context.Context, authAccount *entities.UserAuthAccount) error
	CreateSession(ctx context.Context, session *entities.UserSession) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (*entities.UserSession, error)
	RevokeSession(ctx context.Context, tokenHash string) error
	RevokeAllSessions(ctx context.Context, userID string) error
	UpdateUser(ctx context.Context, user *entities.User) error
}
