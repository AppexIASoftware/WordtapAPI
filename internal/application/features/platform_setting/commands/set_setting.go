package commands

import (
	"context"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type SetPlatformSettingCommand struct {
	Key   string
	Value string
}

type SetPlatformSettingHandler struct {
	repo repositories.PlatformSettingRepository
}

func NewSetPlatformSettingHandler(repo repositories.PlatformSettingRepository) *SetPlatformSettingHandler {
	return &SetPlatformSettingHandler{repo: repo}
}

func (h *SetPlatformSettingHandler) Handle(ctx context.Context, cmd SetPlatformSettingCommand) error {
	key := strings.TrimSpace(cmd.Key)
	if key == "" {
		return apperrors.New("La clave de configuración es obligatoria.")
	}

	return h.repo.Set(ctx, key, strings.TrimSpace(cmd.Value))
}
