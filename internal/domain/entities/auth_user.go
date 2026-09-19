package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccessTier define los niveles de membresía del usuario (free, course, subscription, admin).
type AccessTier string

const (
	AccessTierFree         AccessTier = "free"
	AccessTierCourse       AccessTier = "course"
	AccessTierSubscription AccessTier = "subscription"
	AccessTierAdmin        AccessTier = "admin"
)

// User representa el perfil principal y cuenta del usuario.
type User struct {
	ID                string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	Email             string     `gorm:"type:varchar(320);uniqueIndex;not null" json:"email"`
	Name              string     `gorm:"type:varchar(120);not null" json:"name"`
	PasswordHash      *string    `gorm:"type:text" json:"-"`
	AvatarURL         *string    `gorm:"type:text" json:"avatar_url"`
	AccessTier        AccessTier `gorm:"type:varchar(20);default:'free';not null" json:"access_tier"`
	PreferredLanguage string     `gorm:"type:varchar(10);default:'es';not null" json:"preferred_language"`
	LearningLevel     string     `gorm:"type:varchar(30);default:'beginner';not null" json:"learning_level"`
	Timezone          string     `gorm:"type:varchar(80);default:'UTC';not null" json:"timezone"`
	EmailVerifiedAt   *time.Time `json:"email_verified_at"`
	IsActive          bool       `gorm:"default:true;not null" json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Relaciones
	LearningStats          *UserLearningStats      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"learning_stats,omitempty"`
	NotificationPreference *NotificationPreference `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"notification_preference,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}

// UserAuthAccount vincula cuentas de proveedores OAuth (Google, Apple) con el usuario.
type UserAuthAccount struct {
	ID                string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID            string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_provider_account" json:"user_id"`
	Provider          string    `gorm:"type:varchar(40);not null;uniqueIndex:idx_provider_account" json:"provider"`
	ProviderAccountID string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_provider_account" json:"provider_account_id"`
	CreatedAt         time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (a *UserAuthAccount) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}

// UserSession almacena sesiones activas y tokens para validación y revocación remota.
type UserSession struct {
	ID               string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID           string     `gorm:"type:varchar(36);not null;index" json:"user_id"`
	SessionTokenHash string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	ExpiresAt        time.Time  `gorm:"index;not null" json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	IPAddress        *string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent        *string    `gorm:"type:text" json:"user_agent"`
	CreatedAt        time.Time  `json:"created_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (s *UserSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

// AccountToken almacena tokens temporales de verificación de email y recuperación de contraseña.
type AccountToken struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    *string    `gorm:"type:varchar(36);index" json:"user_id"`
	TokenHash string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	TokenType string     `gorm:"type:varchar(30);not null" json:"token_type"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (t *AccountToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// UserDevice registra dispositivos del usuario y tokens push (FCM/APNs).
type UserDevice struct {
	ID         string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID     string     `gorm:"type:varchar(36);not null;index" json:"user_id"`
	DeviceName *string    `gorm:"type:varchar(120)" json:"device_name"`
	Platform   *string    `gorm:"type:varchar(40)" json:"platform"`
	PushToken  *string    `gorm:"type:varchar(512);uniqueIndex" json:"push_token"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	CreatedAt  time.Time  `json:"created_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (d *UserDevice) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	return nil
}
