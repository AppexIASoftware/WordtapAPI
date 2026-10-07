package queries

import (
	"context"
	"sync"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type PublicSettingsDTO struct {
	ContactEmail string `json:"contact_email"`
	CompanyName  string `json:"company_name"`
	SupportURL   string `json:"support_url"`
}

type GetPublicSettingsHandler struct {
	repo      repositories.PlatformSettingRepository
	mu        sync.RWMutex
	cache     *PublicSettingsDTO
	expiresAt time.Time
}

func NewGetPublicSettingsHandler(repo repositories.PlatformSettingRepository) *GetPublicSettingsHandler {
	return &GetPublicSettingsHandler{repo: repo}
}

func (h *GetPublicSettingsHandler) Handle(ctx context.Context) (*PublicSettingsDTO, error) {
	h.mu.RLock()
	if h.cache != nil && time.Now().Before(h.expiresAt) {
		cached := *h.cache
		h.mu.RUnlock()
		return &cached, nil
	}
	h.mu.RUnlock()

	settings, err := h.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := &PublicSettingsDTO{
		ContactEmail: "soporte@wordtap.app",
		CompanyName:  "Wordtap Studio",
		SupportURL:   "https://wordtap.app/soporte",
	}

	for _, s := range settings {
		switch s.Key {
		case "contact_email":
			if s.Value != "" {
				result.ContactEmail = s.Value
			}
		case "company_name":
			if s.Value != "" {
				result.CompanyName = s.Value
			}
		case "support_url":
			if s.Value != "" {
				result.SupportURL = s.Value
			}
		}
	}

	h.mu.Lock()
	h.cache = result
	h.expiresAt = time.Now().Add(2 * time.Minute)
	h.mu.Unlock()

	return result, nil
}
