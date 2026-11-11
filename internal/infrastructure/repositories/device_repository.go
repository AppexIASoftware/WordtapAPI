package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type MySQLDeviceRepository struct {
	db *gorm.DB
}

func NewMySQLDeviceRepository(db *gorm.DB) domainRepo.DeviceRepository {
	return &MySQLDeviceRepository{db: db}
}

func (r *MySQLDeviceRepository) RegisterDevice(ctx context.Context, device *entities.UserDevice) error {
	now := time.Now()
	device.LastSeenAt = &now
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "push_token"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "device_name", "platform", "last_seen_at"}),
	}).Create(device).Error
}

func (r *MySQLDeviceRepository) RevokeToken(ctx context.Context, userID, pushToken string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND push_token = ?", userID, pushToken).Delete(&entities.UserDevice{}).Error
}

func (r *MySQLDeviceRepository) GetTokensByUser(ctx context.Context, userID string) ([]string, error) {
	var tokens []string
	err := r.db.WithContext(ctx).Model(&entities.UserDevice{}).
		Where("user_id = ? AND push_token IS NOT NULL AND push_token != ''", userID).
		Pluck("push_token", &tokens).Error
	return tokens, err
}

func (r *MySQLDeviceRepository) GetTokensByUsers(ctx context.Context, userIDs []string) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var tokens []string
	err := r.db.WithContext(ctx).Model(&entities.UserDevice{}).
		Where("user_id IN ? AND push_token IS NOT NULL AND push_token != ''", userIDs).
		Pluck("push_token", &tokens).Error
	return tokens, err
}

func (r *MySQLDeviceRepository) GetAllTokens(ctx context.Context) ([]string, error) {
	var tokens []string
	err := r.db.WithContext(ctx).Model(&entities.UserDevice{}).
		Where("push_token IS NOT NULL AND push_token != ''").
		Pluck("push_token", &tokens).Error
	return tokens, err
}

func (r *MySQLDeviceRepository) CountActiveTokens(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.UserDevice{}).
		Where("push_token IS NOT NULL AND push_token != ''").
		Count(&count).Error
	return count, err
}
