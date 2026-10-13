package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Classroom representa un aula virtual privada creada por un docente para una cohorte.
type Classroom struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TeacherID   string    `gorm:"type:varchar(36);index;not null" json:"teacher_id"`
	CourseID    *string   `gorm:"type:varchar(36);index" json:"course_id,omitempty"`
	Name        string    `gorm:"type:varchar(180);not null" json:"name"`
	Code        string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"code"`
	Level           string    `gorm:"type:varchar(50);default:'B1 Intermediate';not null" json:"level"`
	Description     *string   `gorm:"type:text" json:"description,omitempty"`
	AssignedCourses *string   `gorm:"type:text" json:"assigned_courses,omitempty"` // JSON o lista separada por comas de IDs de cursos
	ActiveGames     *string   `gorm:"type:text" json:"active_games,omitempty"`     // JSON o lista de modos de minijuegos habilitados
	WeeklyGoalWords int       `gorm:"default:50;not null" json:"weekly_goal_words"`
	IsActive        bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relaciones
	Teacher     *User                 `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Course      *Course               `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	Enrollments []ClassroomEnrollment `gorm:"foreignKey:ClassroomID;constraint:OnDelete:CASCADE" json:"enrollments,omitempty"`
	Stumbles    []StudentStumble      `gorm:"foreignKey:ClassroomID;constraint:OnDelete:CASCADE" json:"stumbles,omitempty"`
}

func (c *Classroom) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

// ClassroomEnrollment registra la vinculación de un estudiante a un aula privada mediante código.
type ClassroomEnrollment struct {
	ID            string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ClassroomID   string     `gorm:"type:varchar(36);index;not null" json:"classroom_id"`
	StudentID     string     `gorm:"type:varchar(36);index;not null" json:"student_id"`
	EnrolledAt    time.Time  `json:"enrolled_at"`
	LastActiveAt  *time.Time `json:"last_active_at,omitempty"`
	WordsMastered int        `gorm:"default:0;not null" json:"words_mastered"`
	AccuracyPct   int        `gorm:"default:0;not null" json:"accuracy_pct"`
	StreakDays    int        `gorm:"default:0;not null" json:"streak_days"`

	// Relaciones
	Classroom *Classroom `gorm:"foreignKey:ClassroomID" json:"classroom,omitempty"`
	Student   *User      `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

func (e *ClassroomEnrollment) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.EnrolledAt.IsZero() {
		e.EnrolledAt = time.Now()
	}
	return nil
}

// StudentStumble registra términos o reglas de alta dificultad pedagógica dentro del aula.
type StudentStumble struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	ClassroomID   string    `gorm:"type:varchar(36);index;not null" json:"classroom_id"`
	Term          string    `gorm:"type:varchar(150);not null" json:"term"`
	Translation   string    `gorm:"type:varchar(200);not null" json:"translation"`
	FailRate      int       `gorm:"default:40;not null" json:"fail_rate"` // Porcentaje de fallo (ej. 68 para 68%)
	TotalAttempts int       `gorm:"default:0;not null" json:"total_attempts"`
	Notes         *string   `gorm:"type:text" json:"notes,omitempty"`
	GrammarType   string    `gorm:"type:varchar(80);default:'General';not null" json:"grammar_type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Classroom *Classroom `gorm:"foreignKey:ClassroomID" json:"classroom,omitempty"`
}

func (s *StudentStumble) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}
