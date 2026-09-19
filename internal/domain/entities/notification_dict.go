package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType clasifica el propósito del mensaje (daily_word, reminder, streak_risk, system, promotion).
type NotificationType string

const (
	NotificationTypeDailyWord   NotificationType = "daily_word"
	NotificationTypeReminder    NotificationType = "reminder"
	NotificationTypeStreakRisk  NotificationType = "streak_risk"
	NotificationTypeSystem      NotificationType = "system"
	NotificationTypePromotion   NotificationType = "promotion"
)

// TranslationEntry almacena pares de traducción normalizados en español e inglés.
type TranslationEntry struct {
	ID             string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	SourceLanguage string    `gorm:"type:varchar(10);default:'es';not null;uniqueIndex:idx_trans_pair,priority:1" json:"source_language"`
	TargetLanguage string    `gorm:"type:varchar(10);default:'en';not null;uniqueIndex:idx_trans_pair,priority:2" json:"target_language"`
	SourceText     string    `gorm:"type:varchar(500);not null;uniqueIndex:idx_trans_pair,priority:3" json:"source_text"`
	TranslatedText string    `gorm:"type:varchar(500);not null" json:"translated_text"`
	Source         string    `gorm:"type:varchar(30);default:'local';not null" json:"source"`
	CreatedAt      time.Time `json:"created_at"`
}

func (te *TranslationEntry) BeforeCreate(tx *gorm.DB) error {
	if te.ID == "" {
		te.ID = uuid.NewString()
	}
	return nil
}

// UserSearchHistory guarda los términos consultados por el usuario en el diccionario.
type UserSearchHistory struct {
	ID                 string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID             string    `gorm:"type:varchar(36);not null;index:idx_hist_user,priority:1" json:"user_id"`
	QueryText          string    `gorm:"type:varchar(500);not null" json:"query_text"`
	TranslationEntryID *string   `gorm:"type:varchar(36);index" json:"translation_entry_id"`
	SearchedAt         time.Time `gorm:"index:idx_hist_user,priority:2" json:"searched_at"`

	User             *User             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	TranslationEntry *TranslationEntry `gorm:"foreignKey:TranslationEntryID;constraint:OnDelete:SET NULL" json:"translation_entry,omitempty"`
}

func (sh *UserSearchHistory) BeforeCreate(tx *gorm.DB) error {
	if sh.ID == "" {
		sh.ID = uuid.NewString()
	}
	return nil
}

// Notification representa la bandeja de entrada de notificaciones y recordatorios del usuario.
type Notification struct {
	ID               string           `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID           string           `gorm:"type:varchar(36);not null;index:idx_notif_user,priority:1" json:"user_id"`
	NotificationType NotificationType `gorm:"type:varchar(40);not null" json:"notification_type"`
	Title            string           `gorm:"type:varchar(180);not null" json:"title"`
	Body             string           `gorm:"type:text;not null" json:"body"`
	ActionURL        *string          `gorm:"type:text" json:"action_url"`
	VocabularyItemID *string          `gorm:"type:varchar(36);index" json:"vocabulary_item_id"`
	IsRead           bool             `gorm:"default:false;not null;index:idx_notif_user,priority:2" json:"is_read"`
	SentAt           *time.Time       `json:"sent_at"`
	ReadAt           *time.Time       `json:"read_at"`
	CreatedAt        time.Time        `gorm:"index:idx_notif_user,priority:3" json:"created_at"`

	User           *User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	VocabularyItem *VocabularyItem `gorm:"foreignKey:VocabularyItemID;constraint:OnDelete:SET NULL" json:"vocabulary_item,omitempty"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}

// NotificationPreference guarda preferencias horarias y activación de recordatorios del usuario.
type NotificationPreference struct {
	UserID           string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	Enabled          bool      `gorm:"default:true;not null" json:"enabled"`
	DailyWordEnabled bool      `gorm:"default:true;not null" json:"daily_word_enabled"`
	ReminderEnabled  bool      `gorm:"default:true;not null" json:"reminder_enabled"`
	ReminderTime     string    `gorm:"type:time;default:'09:00:00';not null" json:"reminder_time"`
	Timezone         string    `gorm:"type:varchar(80);default:'UTC';not null" json:"timezone"`
	UpdatedAt        time.Time `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// NotificationSchedule permite configurar múltiples horarios de recordatorio semanales.
type NotificationSchedule struct {
	ID               string           `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID           string           `gorm:"type:varchar(36);not null;index" json:"user_id"`
	NotificationType NotificationType `gorm:"type:varchar(40);not null" json:"notification_type"`
	LocalTime        string           `gorm:"type:time;not null" json:"local_time"`
	DaysOfWeek       string           `gorm:"type:varchar(30);default:'1,2,3,4,5,6,7';not null" json:"days_of_week"`
	IsEnabled        bool             `gorm:"default:true;not null" json:"is_enabled"`
	CreatedAt        time.Time        `json:"created_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (ns *NotificationSchedule) BeforeCreate(tx *gorm.DB) error {
	if ns.ID == "" {
		ns.ID = uuid.NewString()
	}
	return nil
}
