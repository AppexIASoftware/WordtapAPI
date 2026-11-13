package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type stubPricingSettingsRepo struct {
	values map[string]string
}

func (s *stubPricingSettingsRepo) Get(_ context.Context, key string) (*entities.PlatformSetting, error) {
	v, ok := s.values[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &entities.PlatformSetting{Key: key, Value: v}, nil
}

func (s *stubPricingSettingsRepo) Set(_ context.Context, key string, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *stubPricingSettingsRepo) GetAll(_ context.Context) ([]entities.PlatformSetting, error) {
	return nil, nil
}

func TestValidateTeacherCoursePricingForRole(t *testing.T) {
	ctx := context.Background()

	t.Run("default blocks free courses and allows paid courses >= minimum", func(t *testing.T) {
		if err := validateTeacherCoursePricing(ctx, nil, entities.AccessTierFree, 0); err == nil {
			t.Fatal("expected free course to be rejected by default")
		}
		if err := validateTeacherCoursePricing(ctx, nil, entities.AccessTierCourse, 2000); err != nil {
			t.Fatalf("expected valid paid course price, got %v", err)
		}
	})

	t.Run("allows free course when courses:author_free_courses is enabled", func(t *testing.T) {
		repo := &stubPricingSettingsRepo{
			values: map[string]string{
				access.RBACSettingKey: `{"courses:author_free_courses":{"instructor":true}}`,
			},
		}
		if err := validateTeacherCoursePricing(ctx, repo, entities.AccessTierFree, 0); err != nil {
			t.Fatalf("expected free course to be allowed when toggle enabled, got %v", err)
		}
		if err := validateTeacherCoursePricing(ctx, repo, entities.AccessTierFree, 500); err == nil {
			t.Fatal("expected non-zero price on free tier to fail")
		}
	})

	t.Run("blocks custom paid course price when monetization:custom_course_price is disabled", func(t *testing.T) {
		repo := &stubPricingSettingsRepo{
			values: map[string]string{
				access.RBACSettingKey: `{"monetization:custom_course_price":{"instructor":false}}`,
			},
		}
		if err := validateTeacherCoursePricing(ctx, repo, entities.AccessTierCourse, 2500); err == nil {
			t.Fatal("expected paid course to be rejected when custom_course_price is disabled")
		}
	})
}
