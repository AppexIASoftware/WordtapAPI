package repositories

import (
	"context"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"gorm.io/gorm"
)

type MySQLNotificationInbox struct{ db *gorm.DB }

func NewMySQLNotificationInbox(db *gorm.DB) *MySQLNotificationInbox {
	return &MySQLNotificationInbox{db: db}
}

func (r *MySQLNotificationInbox) List(ctx context.Context, userID string, limit int, cursor *inbox.Cursor) (inbox.Page, error) {
	page := inbox.Page{Items: []entities.Notification{}}
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if cursor != nil {
		query = query.Where("(created_at < ? OR (created_at = ? AND id < ?))", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	err := query.Order("created_at DESC").Order("id DESC").Limit(limit + 1).Find(&page.Items).Error
	if err != nil {
		return inbox.Page{}, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = inbox.EncodeCursor(inbox.Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	return page, nil
}

func (r *MySQLNotificationInbox) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}

func (r *MySQLNotificationInbox) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&entities.Notification{}).
		Where("user_id = ? AND id = ? AND is_read = ?", userID, id, false).
		Updates(map[string]any{"is_read": true, "read_at": time.Now().UTC()})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		var notif entities.Notification
		if err := r.db.WithContext(ctx).Select("title", "created_at").First(&notif, "id = ?", id).Error; err == nil {
			windowStart := notif.CreatedAt.Add(-10 * time.Minute)
			windowEnd := notif.CreatedAt.Add(10 * time.Minute)
			_ = r.db.WithContext(ctx).Exec(`
				UPDATE classroom_notifications
				SET read_count = read_count + 1,
				    status = CASE WHEN status = 'partial' OR status = 'failed' THEN 'sent' ELSE status END
				WHERE title = ? AND sent_at BETWEEN ? AND ?
			`, notif.Title, windowStart, windowEnd).Error
		}
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.Notification{}).Where("user_id = ? AND id = ?", userID, id).Count(&count).Error
	return count > 0, err
}

func (r *MySQLNotificationInbox) Create(ctx context.Context, n *entities.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *MySQLNotificationInbox) CreateBatch(ctx context.Context, notifs []entities.Notification) error {
	if len(notifs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(&notifs, 100).Error
}

