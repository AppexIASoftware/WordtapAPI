package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type ReviewCalibrationRepository interface {
	GetByUserID(ctx context.Context, userID string) (*entities.TeacherReviewCalibration, error)
	Upsert(ctx context.Context, calibration *entities.TeacherReviewCalibration) error
}
