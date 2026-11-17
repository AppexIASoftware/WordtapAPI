package commands

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

const maxSettingValueBytes = 16384

var allowedPlatformSettingKeys = map[string]bool{
	"contact_email":        true,
	"company_name":         true,
	"support_url":          true,
	"course_min_price_pen": true,
	access.RBACSettingKey:  true,
}

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
	if !allowedPlatformSettingKeys[key] {
		return apperrors.New("Clave de configuración no permitida.")
	}

	val := strings.TrimSpace(cmd.Value)
	if len(val) > maxSettingValueBytes {
		return apperrors.New("El valor de configuración excede el límite permitido.")
	}

	if key == access.RBACSettingKey && val != "" {
		var overrides map[string]map[string]bool
		if err := json.Unmarshal([]byte(val), &overrides); err != nil {
			return apperrors.New("El formato de la matriz de permisos no es válido.")
		}
	}

	return h.repo.Set(ctx, key, val)
}
