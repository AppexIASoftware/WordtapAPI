package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LessonGame links a game to a course lesson.
type LessonGame struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	LessonID      string    `gorm:"type:varchar(36);not null;uniqueIndex:uq_lesson_games,priority:1;index:idx_lesson_games_order,priority:1" json:"lesson_id"`
	TeacherGameID string    `gorm:"type:varchar(36);not null;uniqueIndex:uq_lesson_games,priority:2;index" json:"teacher_game_id"`
	SortOrder     int       `gorm:"default:0;not null;index:idx_lesson_games_order,priority:2" json:"sort_order"`
	IsRequired    bool      `gorm:"default:true;not null" json:"is_required"`
	CreatedAt     time.Time `json:"created_at"`

	// LevelWarning is ephemeral and computed on demand.
	LevelWarning string `gorm:"-" json:"level_warning,omitempty"`

	Lesson      *Lesson      `gorm:"foreignKey:LessonID;constraint:OnDelete:CASCADE" json:"lesson,omitempty"`
	TeacherGame *TeacherGame `gorm:"foreignKey:TeacherGameID;constraint:OnDelete:CASCADE" json:"game,omitempty"`
}

func (lg *LessonGame) BeforeCreate(tx *gorm.DB) error {
	if lg.ID == "" {
		lg.ID = uuid.NewString()
	}
	return nil
}
