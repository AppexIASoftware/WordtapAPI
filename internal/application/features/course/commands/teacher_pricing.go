package commands

import (
	"context"
	"fmt"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

func validateTeacherCoursePricing(ctx context.Context, settings repositories.PlatformSettingRepository, tier entities.AccessTier, price int) error {
	if tier != entities.AccessTierCourse {
		return apperrors.New("Los cursos de docentes requieren un pago único por curso.")
	}
	minimum := resolveMinPriceCents(ctx, settings)
	if price < minimum {
		return apperrors.New(fmt.Sprintf("El precio mínimo para cursos individuales de docentes es de S/ %.2f PEN.", float64(minimum)/100.0))
	}
	return nil
}
