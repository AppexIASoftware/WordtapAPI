package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type PlatformSettingRepository interface {
	Get(ctx context.Context, key string) (*entities.PlatformSetting, error)
	Set(ctx context.Context, key string, value string) error
	GetAll(ctx context.Context) ([]entities.PlatformSetting, error)
}
