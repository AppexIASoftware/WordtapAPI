package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType classifies the message category (daily_word, reminder, streak_risk, system, promotion).
type NotificationType string

const (
	NotificationTypeDailyWord   NotificationType = "daily_word"
	NotificationTypeReminder    NotificationType = "reminder"
	NotificationTypeStreakRisk  NotificationType = "streak_risk"
	NotificationTypeSystem      NotificationType = "system"
	NotificationTypePromotion   NotificationType = "promotion"
)

// TranslationEntry stores normalized Spanish-English translation pairs.
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

// UserSearchHistory stores dictionary search queries performed by users.
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

// Notification represents an inbox item or reminder delivered to a user.
type Notification struct {
	ID                      string           `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID                  string           `gorm:"type:varchar(36);not null;index:idx_notif_user,priority:1" json:"user_id"`
	ClassroomNotificationID *string          `gorm:"type:varchar(36);index" json:"classroom_notification_id,omitempty"`
	NotificationType        NotificationType `gorm:"type:varchar(40);not null" json:"notification_type"`
	Title                   string           `gorm:"type:varchar(180);not null" json:"title"`
	Body                    string           `gorm:"type:text;not null" json:"body"`
	ActionURL               *string          `gorm:"type:text" json:"action_url"`
	VocabularyItemID        *string          `gorm:"type:varchar(36);index" json:"vocabulary_item_id"`
	IsRead                  bool             `gorm:"default:false;not null;index:idx_notif_user,priority:2" json:"is_read"`
	SentAt                  *time.Time       `json:"sent_at"`
	ReadAt                  *time.Time       `json:"read_at"`
	CreatedAt               time.Time        `gorm:"index:idx_notif_user,priority:3" json:"created_at"`

	User                  *User                  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	ClassroomNotification *ClassroomNotification `gorm:"foreignKey:ClassroomNotificationID;constraint:OnDelete:SET NULL" json:"classroom_notification,omitempty"`
	VocabularyItem        *VocabularyItem        `gorm:"foreignKey:VocabularyItemID;constraint:OnDelete:SET NULL" json:"vocabulary_item,omitempty"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}

// NotificationPreference stores user delivery schedules and reminder preferences.
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

// NotificationSchedule enables configuring multiple weekly reminder times.
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

// NotificationScheduleMode stores the scheduling mode (random or custom).
type NotificationScheduleMode struct {
	UserID    string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	Mode      string    `gorm:"type:varchar(20);not null;default:'random'" json:"mode"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// ClassroomNotificationStatus defines the delivery status of a broadcast notification.
type ClassroomNotificationStatus string

const (
	ClassroomNotificationStatusSent    ClassroomNotificationStatus = "sent"
	ClassroomNotificationStatusFailed  ClassroomNotificationStatus = "failed"
	ClassroomNotificationStatusPartial ClassroomNotificationStatus = "partial"
)

// ClassroomNotification represents a broadcast sent by a teacher to a classroom.
// It is a single record for history/audit purposes, distinct from per-student Notification inbox entries.
type ClassroomNotification struct {
	ID             string                      `gorm:"type:varchar(36);primaryKey" json:"id"`
	ClassroomID    *string                     `gorm:"type:varchar(36);index:idx_classroom_notif_classroom_sent,priority:1" json:"classroom_id,omitempty"`
	TeacherID      string                      `gorm:"type:varchar(36);not null;index:idx_classroom_notif_teacher_sent,priority:1" json:"teacher_id"`
	Title          string                      `gorm:"type:varchar(180);not null" json:"title"`
	Message        string                      `gorm:"type:text;not null" json:"message"`
	ActionType     string                      `gorm:"type:varchar(30);not null" json:"action_type"` // open_classroom, open_content
	TargetCourseID *string                     `gorm:"type:varchar(36);index:idx_classroom_notif_course_sent,priority:1" json:"target_course_id,omitempty"`
	TargetLessonID *string                     `gorm:"type:varchar(36);index:idx_classroom_notif_lesson_sent,priority:1" json:"target_lesson_id,omitempty"`
	SentAt         time.Time                   `gorm:"index:idx_classroom_notif_classroom_sent,priority:2;index:idx_classroom_notif_teacher_sent,priority:2;index:idx_classroom_notif_course_sent,priority:2;index:idx_classroom_notif_lesson_sent,priority:2" json:"sent_at"`
	RecipientCount int                         `gorm:"default:0;not null" json:"recipient_count"`
	ReadCount      int                         `gorm:"default:0;not null" json:"read_count"`
	Status         ClassroomNotificationStatus `gorm:"type:varchar(20);default:'sent';not null;index:idx_classroom_notif_status_sent,priority:1" json:"status"`
	CreatedAt      time.Time                   `json:"created_at"`

	// Relaciones
	Classroom    *Classroom `gorm:"foreignKey:ClassroomID;constraint:OnDelete:CASCADE" json:"classroom,omitempty"`
	Teacher      *User      `gorm:"foreignKey:TeacherID;constraint:OnDelete:CASCADE" json:"teacher,omitempty"`
	TargetCourse *Course    `gorm:"foreignKey:TargetCourseID;constraint:OnDelete:SET NULL" json:"target_course,omitempty"`
	TargetLesson *Lesson    `gorm:"foreignKey:TargetLessonID;constraint:OnDelete:SET NULL" json:"target_lesson,omitempty"`
}

func (cn *ClassroomNotification) BeforeCreate(tx *gorm.DB) error {
	if cn.ID == "" {
		cn.ID = uuid.NewString()
	}
	if cn.SentAt.IsZero() {
		cn.SentAt = time.Now().UTC()
	}
	return nil
}



