package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeacherGame define una actividad lúdica personalizada creada por un docente a partir de un arquetipo base.
type TeacherGame struct {
	ID               string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TeacherID        string    `gorm:"type:varchar(36);not null;index" json:"teacher_id"`
	CourseID         *string   `gorm:"type:varchar(36);index" json:"course_id,omitempty"`
	BaseArchetype    string    `gorm:"type:varchar(40);not null" json:"base_archetype"` // match, memory, multiple_choice, word_scramble, listening
	Title            string    `gorm:"type:varchar(180);not null" json:"title"`
	Description      *string   `gorm:"type:text" json:"description,omitempty"`
	Level            string    `gorm:"type:varchar(50);default:'B1 Intermediate';not null" json:"level"`
	TimeLimitSeconds int       `gorm:"default:45;not null" json:"time_limit_seconds"`
	AllowedMistakes  int       `gorm:"default:3;not null" json:"allowed_mistakes"` // 0 = muerte súbita, 1-5 vidas
	RewardXP         int       `gorm:"default:20;not null" json:"reward_xp"`
	ItemCount        int       `gorm:"default:6;not null" json:"item_count"`
	SourceType       string    `gorm:"type:varchar(40);default:'course_lesson';not null" json:"source_type"` // course_lesson, vault, custom_list
	SourceID         *string   `gorm:"type:varchar(36)" json:"source_id,omitempty"`
	CustomWordsJSON  *string   `gorm:"type:json" json:"custom_words_json,omitempty"` // pares de vocabulario [{term, translation, audio_url}]
	IsActive         bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Teacher *User   `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Course  *Course `gorm:"foreignKey:CourseID" json:"course,omitempty"`
}

func (tg *TeacherGame) BeforeCreate(tx *gorm.DB) error {
	if tg.ID == "" {
		tg.ID = uuid.NewString()
	}
	return nil
}
