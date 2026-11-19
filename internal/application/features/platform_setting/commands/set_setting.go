package commands

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

const (
	maxSettingValueBytes = 16384
	minSettingPriceCents = 100
	maxSettingPriceCents = 10_000_000
)

const (
	SettingKeyMonetizationPlans = "monetization_plans_config"
	SettingKeyMonetizationAdMob = "monetization_admob_config"
)

var allowedPlatformSettingKeys = map[string]bool{
	"contact_email":             true,
	"company_name":              true,
	"support_url":               true,
	"course_min_price_pen":      true,
	SettingKeyMonetizationPlans: true,
	SettingKeyMonetizationAdMob: true,
	access.RBACSettingKey:       true,
}

type monetizationPlansPayload struct {
	SubPrice      float64                      `json:"subPrice"`
	TrialDays     int                          `json:"trialDays"`
	FreeLives     int                          `json:"freeLives"`
	CourseMin     float64                      `json:"courseMin"`
	CourseMax     float64                      `json:"courseMax"`
	TeacherSplit  int                          `json:"teacherSplit"`
	PlatformSplit int                          `json:"platformSplit"`
	Currency      string                       `json:"currency,omitempty"`
	Gateways      map[string]map[string]string `json:"gateways,omitempty"`
}

type monetizationAdMobPayload struct {
	InterstitialFreq int  `json:"interstitialFreq"`
	BannersEnabled   bool `json:"bannersEnabled"`
	RewardedEnabled  bool `json:"rewardedEnabled"`
}

func decodeStrictJSON(raw string, dst any) error {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected trailing json")
	}
	return nil
}

func priceToCentsBounded(val float64, minCents, maxCents int) (int, bool) {
	if math.IsNaN(val) || math.IsInf(val, 0) || val <= 0 || val > float64(maxCents)/100.0 {
		return 0, false
	}
	cents := int(math.Round(val * 100))
	if cents < minCents || cents > maxCents {
		return 0, false
	}
	return cents, true
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

	if key == SettingKeyMonetizationPlans {
		var cfg monetizationPlansPayload
		if err := decodeStrictJSON(val, &cfg); err != nil {
			return apperrors.New("El formato de la configuración de planes no es válido.")
		}
		if cfg.Currency != "" && !strings.EqualFold(strings.TrimSpace(cfg.Currency), entities.CurrencyPEN) {
			return apperrors.New("La moneda de la plataforma debe ser PEN (Soles).")
		}
		subCents, okSub := priceToCentsBounded(cfg.SubPrice, minSettingPriceCents, maxSettingPriceCents)
		minCents, okMin := priceToCentsBounded(cfg.CourseMin, minSettingPriceCents, maxSettingPriceCents)
		maxCents, okMax := priceToCentsBounded(cfg.CourseMax, minSettingPriceCents, maxSettingPriceCents)
		if !okSub || !okMin || !okMax || maxCents < minCents || subCents <= 0 {
			return apperrors.New("Los precios de suscripción y límites de curso en Soles deben ser válidos.")
		}
		if cfg.TeacherSplit < 0 || cfg.PlatformSplit < 0 || cfg.TeacherSplit+cfg.PlatformSplit != 100 {
			return apperrors.New("La distribución entre docente y plataforma debe sumar exactamente 100%.")
		}
		cfg.Currency = entities.CurrencyPEN
		normalized, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		val = string(normalized)
		if err := h.repo.Set(ctx, "course_min_price_pen", strconv.Itoa(minCents)); err != nil {
			return err
		}
	}

	if key == SettingKeyMonetizationAdMob {
		var admob monetizationAdMobPayload
		if err := decodeStrictJSON(val, &admob); err != nil || admob.InterstitialFreq < 1 {
			return apperrors.New("La frecuencia de anuncios intersticiales debe ser al menos 1.")
		}
	}

	if key == "course_min_price_pen" {
		minCents, err := strconv.Atoi(val)
		if err != nil || minCents < minSettingPriceCents || minCents > maxSettingPriceCents {
			return apperrors.New("El precio mínimo en céntimos PEN debe ser válido.")
		}
	}

	return h.repo.Set(ctx, key, val)
}
