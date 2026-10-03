package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	ApplicationStatusPending   ApplicationStatus = "pending"
	ApplicationStatusApproved  ApplicationStatus = "approved"
	ApplicationStatusRejected  ApplicationStatus = "rejected"
	ApplicationStatusSuspended ApplicationStatus = "suspended"
)

// TeacherApplication gestiona las solicitudes de usuarios que desean ser instructores.
type TeacherApplication struct {
	ID              string            `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID          string            `gorm:"type:varchar(36);not null;index:idx_teacher_app_user" json:"user_id"`
	Status          ApplicationStatus `gorm:"type:varchar(20);default:'pending';not null;index:idx_teacher_app_status_created,priority:1" json:"status"`
	Bio             *string           `gorm:"type:text" json:"bio"`
	Specialty       *string           `gorm:"type:varchar(120)" json:"specialty"`
	ReviewedBy      *string           `gorm:"type:varchar(36);index" json:"reviewed_by"`
	ReviewedAt      *time.Time        `json:"reviewed_at"`
	RejectionReason *string           `gorm:"type:text" json:"rejection_reason"`
	CreatedAt       time.Time         `gorm:"not null;index:idx_teacher_app_status_created,priority:2" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"not null" json:"updated_at"`

	User     *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Reviewer *User `gorm:"foreignKey:ReviewedBy;constraint:OnDelete:SET NULL" json:"reviewer,omitempty"`
}

func (ta *TeacherApplication) BeforeCreate(tx *gorm.DB) error {
	if ta.ID == "" {
		ta.ID = uuid.NewString()
	}
	if ta.Status == "" {
		ta.Status = ApplicationStatusPending
	}
	return nil
}
