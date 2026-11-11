package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationprefs"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type MySQLNotificationPreferences struct{ db *gorm.DB }

func NewMySQLNotificationPreferences(db *gorm.DB) *MySQLNotificationPreferences {
	return &MySQLNotificationPreferences{db: db}
}

// lockNotificationUser serializes concurrent updates for the target user during writes.
func lockNotificationUser(tx *gorm.DB, userID string) error {
	var user entities.User
	return tx.Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", userID).Error
}

func (r *MySQLNotificationPreferences) Get(ctx context.Context, userID string) (*notification.Preferences, error) {
	return readNotificationPreferences(r.db.WithContext(ctx), userID)
}

func (r *MySQLNotificationPreferences) Put(ctx context.Context, userID string, update notification.Update) (result *notification.Preferences, err error) {
	if err := update.Validate(); err != nil {
		return nil, err
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockNotificationUser(tx, userID); err != nil {
			return err
		}
		// Explicit maps preserve false values despite GORM's default:true tags.
		values := map[string]any{"user_id": userID, "enabled": *update.Enabled, "daily_word_enabled": *update.DailyWordEnabled, "reminder_enabled": *update.ReminderEnabled, "timezone": update.Timezone, "reminder_time": "09:00:00"}
		if err := tx.Table("notification_preferences").Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"enabled", "daily_word_enabled", "reminder_enabled", "timezone"})}).Create(values).Error; err != nil {
			return err
		}
		mode := map[string]any{"user_id": userID, "mode": update.Mode}
		if err := tx.Table("notification_schedule_modes").Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"mode"})}).Create(mode).Error; err != nil {
			return err
		}
		if update.Mode == notification.ModeCustom {
			if err := tx.Where("user_id = ? AND notification_type = ?", userID, entities.NotificationTypeReminder).Delete(&entities.NotificationSchedule{}).Error; err != nil {
				return err
			}
			for _, schedule := range update.Schedules {
				days := make([]string, len(schedule.Days))
				for i, day := range schedule.Days {
					days[i] = strconv.Itoa(day)
				}
				row := map[string]any{"id": uuid.NewString(), "user_id": userID, "notification_type": entities.NotificationTypeReminder, "local_time": schedule.Time + ":00", "days_of_week": strings.Join(days, ","), "is_enabled": *schedule.Enabled}
				if err := tx.Table("notification_schedules").Create(row).Error; err != nil {
					return err
				}
			}
		}
		result, err = readNotificationPreferences(tx, userID)
		return err
	})
	return
}

func readNotificationPreferences(tx *gorm.DB, userID string) (*notification.Preferences, error) {
	result := &notification.Preferences{Enabled: true, DailyWordEnabled: true, ReminderEnabled: true, Timezone: "UTC", Mode: notification.ModeRandom, Schedules: []notification.Schedule{}, RandomCount: 3, RandomStart: "07:00", RandomEnd: "22:00"}
	var pref entities.NotificationPreference
	if err := tx.Where("user_id = ?", userID).First(&pref).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		result.Enabled, result.DailyWordEnabled, result.ReminderEnabled, result.Timezone = pref.Enabled, pref.DailyWordEnabled, pref.ReminderEnabled, pref.Timezone
	}
	var rows []entities.NotificationSchedule
	if err := tx.Where("user_id = ? AND notification_type = ?", userID, entities.NotificationTypeReminder).Order("local_time, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		schedule, err := decodeLegacySchedule(row)
		if err != nil {
			return nil, err
		}
		result.Schedules = append(result.Schedules, schedule)
	}
	if len(rows) > 0 {
		result.Mode = notification.ModeCustom
	}
	var mode struct{ Mode string }
	err := tx.Table("notification_schedule_modes").Select("mode").Where("user_id = ?", userID).Take(&mode).Error
	if err == nil {
		if mode.Mode != notification.ModeRandom && mode.Mode != notification.ModeCustom {
			return nil, fmt.Errorf("invalid stored schedule mode")
		}
		result.Mode = mode.Mode
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return result, nil
}

func decodeLegacySchedule(row entities.NotificationSchedule) (notification.Schedule, error) {
	days := []int{}
	for _, item := range strings.Split(row.DaysOfWeek, ",") {
		day, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil || day < 1 || day > 7 {
			return notification.Schedule{}, fmt.Errorf("invalid stored schedule days")
		}
		days = append(days, day)
	}
	t := row.LocalTime
	if len(t) == 8 {
		t = strings.TrimSuffix(t, ":00")
	}
	return notification.Schedule{Time: t, Days: days, Enabled: row.IsEnabled}, nil
}
