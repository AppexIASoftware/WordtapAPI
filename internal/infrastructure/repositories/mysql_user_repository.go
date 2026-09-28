package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type MySQLUserRepository struct {
	db *gorm.DB
}

func NewMySQLUserRepository(db *gorm.DB) domainRepo.UserRepository {
	return &MySQLUserRepository{db: db}
}

func (r *MySQLUserRepository) FindByID(ctx context.Context, id string) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).
		Preload("LearningStats").
		Preload("NotificationPreference").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MySQLUserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).
		Preload("LearningStats").
		Preload("NotificationPreference").
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MySQLUserRepository) FindAuthAccount(ctx context.Context, provider, providerAccountID string) (*entities.UserAuthAccount, error) {
	var account entities.UserAuthAccount
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("User.LearningStats").
		Preload("User.NotificationPreference").
		Where("provider = ? AND provider_account_id = ?", provider, providerAccountID).
		First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *MySQLUserRepository) CreateWithInitialState(ctx context.Context, user *entities.User, authAccount *entities.UserAuthAccount) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		if authAccount != nil {
			authAccount.UserID = user.ID
			if err := tx.Create(authAccount).Error; err != nil {
				return err
			}
		}

		stats := entities.UserLearningStats{
			UserID:            user.ID,
			WordsLearned:      0,
			LessonsCompleted:  0,
			GamesCompleted:    0,
			TotalStars:        0,
			TotalPoints:       0,
			CurrentStreakDays: 0,
			LongestStreakDays: 0,
			AccuracyPercent:   0.0,
		}
		if err := tx.Create(&stats).Error; err != nil {
			return err
		}
		user.LearningStats = &stats

		tz := user.Timezone
		if tz == "" {
			tz = "UTC"
		}
		prefs := entities.NotificationPreference{
			UserID:           user.ID,
			Enabled:          true,
			DailyWordEnabled: true,
			ReminderEnabled:  true,
			ReminderTime:     "09:00:00",
			Timezone:         tz,
		}
		if err := tx.Create(&prefs).Error; err != nil {
			return err
		}
		user.NotificationPreference = &prefs

		return nil
	})
}

func (r *MySQLUserRepository) CreateAuthAccount(ctx context.Context, authAccount *entities.UserAuthAccount) error {
	return r.db.WithContext(ctx).Create(authAccount).Error
}

func (r *MySQLUserRepository) CreateSession(ctx context.Context, session *entities.UserSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *MySQLUserRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*entities.UserSession, error) {
	var session entities.UserSession
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("session_token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *MySQLUserRepository) RevokeSession(ctx context.Context, tokenHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entities.UserSession{}).
		Where("session_token_hash = ?", tokenHash).
		Update("revoked_at", &now).Error
}

func (r *MySQLUserRepository) UpdateUser(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}
