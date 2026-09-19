package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Achievement define las insignias y logros disponibles en la app.
type Achievement struct {
	ID          string  `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code        string  `gorm:"type:varchar(80);uniqueIndex;not null" json:"code"`
	Name        string  `gorm:"type:varchar(150);not null" json:"name"`
	Description string  `gorm:"type:text;not null" json:"description"`
	IconURL     *string `gorm:"type:text" json:"icon_url"`
	Requirement string  `gorm:"type:json;not null" json:"requirement"`
	IsActive    bool    `gorm:"default:true;not null" json:"is_active"`
}

func (a *Achievement) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.Requirement == "" {
		a.Requirement = "{}"
	}
	return nil
}

// UserAchievement registra las insignias desbloqueadas por cada usuario.
type UserAchievement struct {
	UserID        string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	AchievementID string    `gorm:"type:varchar(36);primaryKey" json:"achievement_id"`
	AwardedAt     time.Time `json:"awarded_at"`

	User        *User        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Achievement *Achievement `gorm:"foreignKey:AchievementID;constraint:OnDelete:CASCADE" json:"achievement,omitempty"`
}

// DailyChallenge define los retos diarios para fomentar el hábito de estudio.
type DailyChallenge struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	ChallengeDate time.Time `gorm:"type:date;uniqueIndex;not null" json:"challenge_date"`
	Title         string    `gorm:"type:varchar(180);not null" json:"title"`
	Description   string    `gorm:"type:text;not null" json:"description"`
	Requirement   string    `gorm:"type:json;not null" json:"requirement"`
	RewardPoints  int       `gorm:"default:0;not null" json:"reward_points"`
	IsActive      bool      `gorm:"default:true;not null" json:"is_active"`
}

func (dc *DailyChallenge) BeforeCreate(tx *gorm.DB) error {
	if dc.ID == "" {
		dc.ID = uuid.NewString()
	}
	if dc.Requirement == "" {
		dc.Requirement = "{}"
	}
	return nil
}

// UserDailyChallenge mide el progreso del usuario hacia la meta del desafío del día.
type UserDailyChallenge struct {
	UserID      string     `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	ChallengeID string     `gorm:"type:varchar(36);primaryKey" json:"challenge_id"`
	Progress    int        `gorm:"default:0;not null" json:"progress"`
	CompletedAt *time.Time `json:"completed_at"`

	User           *User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	DailyChallenge *DailyChallenge `gorm:"foreignKey:ChallengeID;constraint:OnDelete:CASCADE" json:"daily_challenge,omitempty"`
}

// LeaderboardScore acumula puntos por periodo (weekly, monthly, all_time) para rankings.
type LeaderboardScore struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID      string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_lead_entry,priority:1" json:"user_id"`
	PeriodType  string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_lead_entry,priority:2;index:idx_lead_period,priority:1" json:"period_type"`
	PeriodStart time.Time `gorm:"type:date;not null;uniqueIndex:idx_lead_entry,priority:3;index:idx_lead_period,priority:2" json:"period_start"`
	Points      int       `gorm:"default:0;not null;index:idx_lead_period,priority:3" json:"points"`
	UpdatedAt   time.Time `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (ls *LeaderboardScore) BeforeCreate(tx *gorm.DB) error {
	if ls.ID == "" {
		ls.ID = uuid.NewString()
	}
	return nil
}

// AdminRole define roles y permisos en formato JSON del panel de administración.
type AdminRole struct {
	ID          string `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(80);uniqueIndex;not null" json:"name"`
	Permissions string `gorm:"type:json;not null" json:"permissions"`
}

func (ar *AdminRole) BeforeCreate(tx *gorm.DB) error {
	if ar.ID == "" {
		ar.ID = uuid.NewString()
	}
	if ar.Permissions == "" {
		ar.Permissions = "{}"
	}
	return nil
}

// UserAdminRole vincula a un usuario con un rol de administración.
type UserAdminRole struct {
	UserID     string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	RoleID     string    `gorm:"type:varchar(36);primaryKey" json:"role_id"`
	AssignedAt time.Time `json:"assigned_at"`

	User *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Role *AdminRole `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"`
}

// AuditLog bitácora de auditoría para trazabilidad de cambios administrativos.
type AuditLog struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	ActorUserID *string   `gorm:"type:varchar(36);index" json:"actor_user_id"`
	Action      string    `gorm:"type:varchar(100);not null" json:"action"`
	EntityType  string    `gorm:"type:varchar(80);not null" json:"entity_type"`
	EntityID    *string   `gorm:"type:varchar(36);index" json:"entity_id"`
	Changes     string    `gorm:"type:json" json:"changes"`
	IPAddress   *string   `gorm:"type:varchar(45)" json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`

	ActorUser *User `gorm:"foreignKey:ActorUserID;constraint:OnDelete:SET NULL" json:"actor_user,omitempty"`
}

func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == "" {
		al.ID = uuid.NewString()
	}
	return nil
}

// ContentReport almacena incidencias o errores en palabras y preguntas reportados por alumnos.
type ContentReport struct {
	ID             string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ReporterUserID *string    `gorm:"type:varchar(36);index" json:"reporter_user_id"`
	LessonID       *string    `gorm:"type:varchar(36);index" json:"lesson_id"`
	GameQuestionID *string    `gorm:"type:varchar(36);index" json:"game_question_id"`
	Reason         string     `gorm:"type:varchar(120);not null" json:"reason"`
	Description    *string    `gorm:"type:text" json:"description"`
	Status         string     `gorm:"type:varchar(30);default:'open';not null" json:"status"`
	ResolvedBy     *string    `gorm:"type:varchar(36);index" json:"resolved_by"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	CreatedAt      time.Time  `json:"created_at"`

	ReporterUser *User         `gorm:"foreignKey:ReporterUserID;constraint:OnDelete:SET NULL" json:"reporter_user,omitempty"`
	Lesson       *Lesson       `gorm:"foreignKey:LessonID;constraint:OnDelete:SET NULL" json:"lesson,omitempty"`
	GameQuestion *GameQuestion `gorm:"foreignKey:GameQuestionID;constraint:OnDelete:SET NULL" json:"game_question,omitempty"`
}

func (cr *ContentReport) BeforeCreate(tx *gorm.DB) error {
	if cr.ID == "" {
		cr.ID = uuid.NewString()
	}
	return nil
}
