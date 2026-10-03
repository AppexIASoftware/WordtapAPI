package entities

import "time"

// PlatformSetting almacena configuraciones dinámicas de la plataforma (clave-valor).
type PlatformSetting struct {
	Key         string    `gorm:"type:varchar(80);primaryKey" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Description *string   `gorm:"type:varchar(255)" json:"description"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}
