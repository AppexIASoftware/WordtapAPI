package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type MySQLReviewCalibrationRepository struct {
	db *gorm.DB
}

func NewMySQLReviewCalibrationRepository(db *gorm.DB) *MySQLReviewCalibrationRepository {
	return &MySQLReviewCalibrationRepository{db: db}
}

func (r *MySQLReviewCalibrationRepository) GetByUserID(ctx context.Context, userID string) (*entities.TeacherReviewCalibration, error) {
	var item entities.TeacherReviewCalibration
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *MySQLReviewCalibrationRepository) Upsert(ctx context.Context, calibration *entities.TeacherReviewCalibration) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"ease_factor", "quality", "repetitions", "updated_at"}),
	}).Create(calibration).Error
}
