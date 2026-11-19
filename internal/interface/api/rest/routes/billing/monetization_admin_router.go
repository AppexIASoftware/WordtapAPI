package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	settingCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/commands"
	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	infraRepo "github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type MonetizationOverviewStore interface {
	ListMonetizationOverviewCourses(ctx context.Context, instructorID string) ([]infraRepo.AdminMonetizationCourseRow, error)
}

type PlansConfigDTO struct {
	SubPrice      float64 `json:"subPrice"`
	TrialDays     int     `json:"trialDays"`
	FreeLives     int     `json:"freeLives"`
	CourseMin     float64 `json:"courseMin"`
	CourseMax     float64 `json:"courseMax"`
	TeacherSplit  int     `json:"teacherSplit"`
	PlatformSplit int     `json:"platformSplit"`
	Currency      string  `json:"currency"`
	Gateways      any     `json:"gateways,omitempty"`
}

type AdMobConfigDTO struct {
	InterstitialFreq int  `json:"interstitialFreq"`
	BannersEnabled   bool `json:"bannersEnabled"`
	RewardedEnabled  bool `json:"rewardedEnabled"`
}

type MonetizationAdminRouter struct {
	store       MonetizationOverviewStore
	settingRepo domainRepo.PlatformSettingRepository
}

func NewMonetizationAdminRouter(store MonetizationOverviewStore, settingRepo domainRepo.PlatformSettingRepository) *MonetizationAdminRouter {
	return &MonetizationAdminRouter{store: store, settingRepo: settingRepo}
}

func (r *MonetizationAdminRouter) RegisterRoutes(v1 *echo.Group, authMiddleware, instructorOrAdmin echo.MiddlewareFunc) {
	v1.GET("/admin/monetization/overview", r.GetOverview, authMiddleware, instructorOrAdmin)
}

func (r *MonetizationAdminRouter) GetOverview(c *echo.Context) error {
	rawUserID, _ := c.Get("user_id").(string)
	userID := strings.TrimSpace(rawUserID)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	roleStr, _ := c.Get("user_role").(string)
	role := entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr)))

	scopedTeacherID := ""
	if role == entities.RoleInstructor {
		scopedTeacherID = userID
	}

	ctx := c.Request().Context()
	plans, admob := loadMonetizationConfigs(ctx, r.settingRepo, role == entities.RoleAdmin)

	courses, err := r.store.ListMonetizationOverviewCourses(ctx, scopedTeacherID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el resumen de monetización.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"currency":     entities.CurrencyPEN,
		"plans_config": plans,
		"admob_config": admob,
		"courses":      courses,
	})
}

func loadMonetizationConfigs(ctx context.Context, repo domainRepo.PlatformSettingRepository, isAdmin bool) (PlansConfigDTO, AdMobConfigDTO) {
	plans := PlansConfigDTO{
		SubPrice:      29.90,
		TrialDays:     7,
		FreeLives:     5,
		CourseMin:     20.00,
		CourseMax:     199.90,
		TeacherSplit:  70,
		PlatformSplit: 30,
		Currency:      entities.CurrencyPEN,
	}
	if isAdmin {
		plans.Gateways = map[string]any{
			"mercadopago": map[string]string{"planId": "mp_wordtap_pro_pen"},
			"stripe":      map[string]string{"subPriceId": "price_wordtap_pro_pen"},
			"apple":       map[string]string{"productId": "com.wordtap.pro.monthly", "tier": "S/ 29.90/mes"},
			"google":      map[string]string{"productId": "wordtap_pro_monthly", "basePlanId": "monthly-pen"},
		}
	}
	admob := AdMobConfigDTO{
		InterstitialFreq: 3,
		BannersEnabled:   true,
		RewardedEnabled:  true,
	}
	if repo == nil {
		return plans, admob
	}

	settings, err := repo.GetAll(ctx)
	if err != nil {
		return plans, admob
	}

	hasPlansJSON := false
	minPriceCents := 0
	for _, s := range settings {
		switch s.Key {
		case settingCmd.SettingKeyMonetizationPlans:
			var parsed PlansConfigDTO
			if err := json.Unmarshal([]byte(s.Value), &parsed); err == nil && parsed.SubPrice > 0 {
				if parsed.Gateways == nil {
					parsed.Gateways = plans.Gateways
				}
				if !isAdmin {
					parsed.Gateways = nil
				}
				parsed.Currency = entities.CurrencyPEN
				plans = parsed
				hasPlansJSON = true
			}
		case settingCmd.SettingKeyMonetizationAdMob:
			var parsed AdMobConfigDTO
			if err := json.Unmarshal([]byte(s.Value), &parsed); err == nil && parsed.InterstitialFreq >= 1 {
				admob = parsed
			}
		case courseCmd.CourseMinPriceSettingKey:
			if v, err := strconv.Atoi(strings.TrimSpace(s.Value)); err == nil && v > 0 {
				minPriceCents = v
			}
		}
	}
	if !hasPlansJSON && minPriceCents > 0 {
		plans.CourseMin = float64(minPriceCents) / 100.0
	}
	return plans, admob
}
