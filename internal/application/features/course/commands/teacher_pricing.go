package commands

import (
	"context"
	"fmt"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

func validateTeacherCoursePricing(ctx context.Context, settings repositories.PlatformSettingRepository, tier entities.AccessTier, price int) error {
	return validateTeacherCoursePricingForRole(ctx, settings, entities.RoleInstructor, tier, price)
}

func validateTeacherCoursePricingForRole(ctx context.Context, settings repositories.PlatformSettingRepository, role entities.UserRole, tier entities.AccessTier, price int) error {
	if role == "" {
		role = entities.RoleInstructor
	}
	eval := access.NewRBACPolicyEvaluator(settings)

	if tier == entities.AccessTierFree {
		if eval.IsAllowed(ctx, role, access.PermCoursesAuthorFreeCourses) {
			if price != 0 {
				return apperrors.New("Los cursos gratuitos deben tener precio 0.")
			}
			return nil
		}
		return apperrors.New("Los cursos de docentes requieren un pago único por curso.")
	}

	if tier != entities.AccessTierCourse {
		return apperrors.New("Los cursos de docentes requieren un pago único por curso.")
	}

	if !eval.IsAllowed(ctx, role, access.PermMonetizationCustomCoursePrice) {
		return apperrors.New("Tu rol no tiene permiso para fijar precios personalizados en cursos.")
	}

	minimum := resolveMinPriceCents(ctx, settings)
	if price < minimum {
		return apperrors.New(fmt.Sprintf("El precio mínimo para cursos individuales de docentes es de S/ %.2f PEN.", float64(minimum)/100.0))
	}
	return nil
}
