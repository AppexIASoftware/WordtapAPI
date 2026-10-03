package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type GetAllSettingsHandler struct {
	repo repositories.PlatformSettingRepository
}

func NewGetAllSettingsHandler(repo repositories.PlatformSettingRepository) *GetAllSettingsHandler {
	return &GetAllSettingsHandler{repo: repo}
}

func (h *GetAllSettingsHandler) Handle(ctx context.Context) ([]entities.PlatformSetting, error) {
	return h.repo.GetAll(ctx)
}
