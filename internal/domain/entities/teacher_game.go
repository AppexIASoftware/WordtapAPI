package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeacherGame defines a game activity created from a base archetype.
type TeacherGame struct {
	ID               string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TeacherID        string    `gorm:"type:varchar(36);not null;index" json:"teacher_id"`
	CourseID         *string   `gorm:"type:varchar(36);index" json:"course_id,omitempty"`
	CategoryID       *string   `gorm:"type:varchar(36);index" json:"category_id,omitempty"`
	BaseArchetype    string    `gorm:"type:varchar(40);not null" json:"base_archetype"` // match, memory, multiple_choice, word_scramble, listening
	Title            string    `gorm:"type:varchar(180);not null" json:"title"`
	Description      *string   `gorm:"type:text" json:"description,omitempty"`
	Level            string     `gorm:"type:varchar(50);default:'B1 Intermediate';not null" json:"level"`
	AccessTier       AccessTier `gorm:"type:varchar(20);default:'free';not null" json:"access_tier"`
	TimeLimitSeconds int        `gorm:"default:45;not null" json:"time_limit_seconds"`
	AllowedMistakes  int       `gorm:"default:3;not null" json:"allowed_mistakes"` // 0 = sudden death, 1-5 lives
	RewardXP         int       `gorm:"default:20;not null" json:"reward_xp"`
	ItemCount        int       `gorm:"default:6;not null" json:"item_count"`
	SourceType       string    `gorm:"type:varchar(40);default:'course_lesson';not null" json:"source_type"` // course_lesson, vault, custom_list
	SourceID         *string   `gorm:"type:varchar(36)" json:"source_id,omitempty"`
	CustomWordsJSON  *string   `gorm:"type:json" json:"custom_words_json,omitempty"` // vocabulary pairs [{term, translation, audio_url}]
	OwnerType        OwnerType `gorm:"type:varchar(20);default:'teacher';not null;index" json:"owner_type"`
	IsGlobal         bool      `gorm:"default:false;not null;index" json:"is_global"`
	IsPublishedStandalone bool `gorm:"default:false;not null;index" json:"is_published_standalone"`
	AudienceScope    string    `gorm:"type:varchar(30);default:'all_students';not null" json:"audience_scope"`
	AssignedCourseIDs *string  `gorm:"type:text" json:"assigned_course_ids,omitempty"`
	IsActive         bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// UsageCount is ephemeral and counts active lesson links.
	UsageCount int `gorm:"-" json:"usage_count"`

	Teacher  *User            `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Course   *Course          `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	Category *ContentCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

func (tg *TeacherGame) BeforeCreate(tx *gorm.DB) error {
	if tg.ID == "" {
		tg.ID = uuid.NewString()
	}
	if tg.AccessTier == "" {
		tg.AccessTier = AccessTierFree
	}
	if tg.AudienceScope == "" {
		tg.AudienceScope = "all_students"
	}
	return nil
}
