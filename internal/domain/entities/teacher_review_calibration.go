package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeacherReviewCalibration almacena los parámetros de calibración SM-2 configurados por cada docente.
type TeacherReviewCalibration struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID      string    `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	EaseFactor  float64   `gorm:"type:decimal(4,2);default:2.50;not null" json:"ease_factor"`
	Quality     int       `gorm:"default:4;not null" json:"quality"`
	Repetitions int       `gorm:"default:3;not null" json:"repetitions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (c *TeacherReviewCalibration) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}
