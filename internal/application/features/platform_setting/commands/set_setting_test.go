package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type stubSettingRepo struct {
	data map[string]string
}

func (s *stubSettingRepo) Get(_ context.Context, key string) (*entities.PlatformSetting, error) {
	if v, ok := s.data[key]; ok {
		return &entities.PlatformSetting{Key: key, Value: v, UpdatedAt: time.Now().UTC()}, nil
	}
	return nil, nil
}

func (s *stubSettingRepo) Set(_ context.Context, key, value string) error {
	s.data[key] = value
	return nil
}

func (s *stubSettingRepo) GetAll(_ context.Context) ([]entities.PlatformSetting, error) {
	out := make([]entities.PlatformSetting, 0, len(s.data))
	for k, v := range s.data {
		out = append(out, entities.PlatformSetting{Key: k, Value: v, UpdatedAt: time.Now().UTC()})
	}
	return out, nil
}

func TestSetPlatformSetting_MonetizationValidationAndPENNormalization(t *testing.T) {
	repo := &stubSettingRepo{data: map[string]string{}}
	handler := NewSetPlatformSettingHandler(repo)

	t.Run("accepts plans config, normalizes currency to PEN, and syncs course_min_price_pen", func(t *testing.T) {
		payload := `{"subPrice":29.9,"trialDays":7,"freeLives":5,"courseMin":25.5,"courseMax":199.9,"teacherSplit":70,"platformSplit":30,"gateways":{"mercadopago":{"planId":"mp-pro"}}}`
		err := handler.Handle(context.Background(), SetPlatformSettingCommand{
			Key:   SettingKeyMonetizationPlans,
			Value: payload,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.data["course_min_price_pen"] != "2550" {
			t.Fatalf("expected course_min_price_pen=2550, got %q", repo.data["course_min_price_pen"])
		}
		saved := repo.data[SettingKeyMonetizationPlans]
		if !strings.Contains(saved, `"currency":"PEN"`) || !strings.Contains(saved, `"mp-pro"`) {
			t.Fatalf("expected normalized currency PEN and preserved gateways, got %s", saved)
		}
	})

	t.Run("rejects non-PEN currency and invalid split total", func(t *testing.T) {
		badCurrency := `{"subPrice":29.9,"courseMin":20,"courseMax":100,"teacherSplit":70,"platformSplit":30,"currency":"USD"}`
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{Key: SettingKeyMonetizationPlans, Value: badCurrency}); err == nil {
			t.Fatal("expected error when currency is USD")
		}
		badSplit := `{"subPrice":29.9,"courseMin":20,"courseMax":100,"teacherSplit":80,"platformSplit":30}`
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{Key: SettingKeyMonetizationPlans, Value: badSplit}); err == nil {
			t.Fatal("expected error when splits != 100")
		}
		subCentBelowMin := `{"subPrice":0.004,"courseMin":20,"courseMax":100,"teacherSplit":70,"platformSplit":30}`
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{Key: SettingKeyMonetizationPlans, Value: subCentBelowMin}); err == nil {
			t.Fatal("expected error for sub-cent subPrice rounding below 100 cents")
		}
		overflowPrice := `{"subPrice":1e20,"courseMin":20,"courseMax":100,"teacherSplit":70,"platformSplit":30}`
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{Key: SettingKeyMonetizationPlans, Value: overflowPrice}); err == nil {
			t.Fatal("expected error for overflow subPrice")
		}
		unknownKey := `{"subPrice":29.9,"courseMin":20,"courseMax":100,"teacherSplit":70,"platformSplit":30,"injected":"bad"}`
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{Key: SettingKeyMonetizationPlans, Value: unknownKey}); err == nil {
			t.Fatal("expected error for unknown JSON fields")
		}
	})

	t.Run("validates admob interstitial frequency and course_min_price_pen", func(t *testing.T) {
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{
			Key:   SettingKeyMonetizationAdMob,
			Value: `{"interstitialFreq":0,"bannersEnabled":true,"rewardedEnabled":true}`,
		}); err == nil {
			t.Fatal("expected error for interstitialFreq < 1")
		}
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{
			Key:   SettingKeyMonetizationAdMob,
			Value: `{"interstitialFreq":4,"bannersEnabled":true,"rewardedEnabled":false}`,
		}); err != nil {
			t.Fatalf("unexpected error for valid admob config: %v", err)
		}
		if err := handler.Handle(context.Background(), SetPlatformSettingCommand{
			Key:   "course_min_price_pen",
			Value: "-100",
		}); err == nil {
			t.Fatal("expected error for non-positive course_min_price_pen")
		}
	})
}
