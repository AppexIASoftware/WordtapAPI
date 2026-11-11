package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type DeviceRepository interface {
	RegisterDevice(ctx context.Context, device *entities.UserDevice) error
	RevokeToken(ctx context.Context, userID, pushToken string) error
	GetTokensByUser(ctx context.Context, userID string) ([]string, error)
	GetTokensByUsers(ctx context.Context, userIDs []string) ([]string, error)
	GetAllTokens(ctx context.Context) ([]string, error)
	CountActiveTokens(ctx context.Context) (int64, error)
}
