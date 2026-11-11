package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type MySQLPlatformSettingRepository struct {
	db *gorm.DB
}

func NewMySQLPlatformSettingRepository(db *gorm.DB) domainRepo.PlatformSettingRepository {
	return &MySQLPlatformSettingRepository{db: db}
}

func (r *MySQLPlatformSettingRepository) Get(ctx context.Context, key string) (*entities.PlatformSetting, error) {
	var setting entities.PlatformSetting
	if err := r.db.WithContext(ctx).Where("`key` = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *MySQLPlatformSettingRepository) Set(ctx context.Context, key string, value string) error {
	now := time.Now()
	setting := entities.PlatformSetting{
		Key:       key,
		Value:     value,
		UpdatedAt: now,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&setting).Error
}

func (r *MySQLPlatformSettingRepository) GetAll(ctx context.Context) ([]entities.PlatformSetting, error) {
	var settings []entities.PlatformSetting
	if err := r.db.WithContext(ctx).Order("`key` ASC").Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}
