package entities

import (
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserLessonProgress registra el porcentaje y avance del usuario en cada lección.
type UserLessonProgress struct {
	ID              string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID          string     `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_lesson,priority:1;index:idx_prog_user,priority:1" json:"user_id"`
	LessonID        string     `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_lesson,priority:2" json:"lesson_id"`
	CompletedItems  int        `gorm:"default:0;not null" json:"completed_items"`
	TotalItems      int        `gorm:"default:0;not null" json:"total_items"`
	ProgressPercent float64    `gorm:"type:decimal(5,2);default:0.00;not null" json:"progress_percent"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	LastPosition    int        `gorm:"default:0;not null" json:"last_position"`
	UpdatedAt       time.Time  `gorm:"index:idx_prog_user,priority:2" json:"updated_at"`

	User   *User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Lesson *Lesson `gorm:"foreignKey:LessonID;constraint:OnDelete:CASCADE" json:"lesson,omitempty"`
}

func (p *UserLessonProgress) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.CalculateProgress()
	return nil
}

func (p *UserLessonProgress) BeforeUpdate(tx *gorm.DB) error {
	p.CalculateProgress()
	return nil
}

func (p *UserLessonProgress) CalculateProgress() {
	if p.TotalItems > 0 {
		percent := (float64(p.CompletedItems) / float64(p.TotalItems)) * 100.0
		if percent > 100.0 {
			percent = 100.0
		}
		p.ProgressPercent = percent
	} else {
		p.ProgressPercent = 0.0
	}

	if p.ProgressPercent >= 100.0 && p.CompletedAt == nil {
		now := time.Now()
		p.CompletedAt = &now
	}
}

// UserVocabulary rastrea palabras favoritas, difíciles, conteo de aciertos y parámetros SM-2 del usuario.
type UserVocabulary struct {
	ID                string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID            string     `gorm:"type:varchar(36);not null;index:idx_user_vocab,priority:1" json:"user_id"`
	VocabularyItemID  *string    `gorm:"type:varchar(36);index:idx_user_vocab,priority:2" json:"vocabulary_item_id"`
	ContentBankItemID *string    `gorm:"type:varchar(36);index" json:"content_bank_item_id"`
	IsFavorite        bool       `gorm:"default:false;not null" json:"is_favorite"`
	IsDifficult       bool       `gorm:"default:false;not null" json:"is_difficult"`
	TimesSeen         int        `gorm:"default:0;not null" json:"times_seen"`
	TimesCorrect      int        `gorm:"default:0;not null" json:"times_correct"`
	EaseFactor        float64    `gorm:"type:decimal(3,2);default:2.50;not null" json:"ease_factor"`
	IntervalDays      int        `gorm:"default:1;not null" json:"interval_days"`
	RepetitionNumber  int        `gorm:"default:0;not null" json:"repetition_number"`
	NextReviewDate    *time.Time `gorm:"index" json:"next_review_date"`
	LastReviewedAt    *time.Time `json:"last_reviewed_at"`

	User            *User            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	VocabularyItem  *VocabularyItem  `gorm:"foreignKey:VocabularyItemID;constraint:OnDelete:CASCADE" json:"vocabulary_item,omitempty"`
	ContentBankItem *ContentBankItem `gorm:"foreignKey:ContentBankItemID;constraint:OnDelete:SET NULL" json:"content_bank_item,omitempty"`
}

func (uv *UserVocabulary) BeforeCreate(tx *gorm.DB) error {
	if uv.ID == "" {
		uv.ID = uuid.NewString()
	}
	if uv.EaseFactor <= 0 {
		uv.EaseFactor = 2.50
	}
	if uv.IntervalDays <= 0 {
		uv.IntervalDays = 1
	}
	return nil
}

// ApplySM2 actualiza los parámetros de repetición espaciada según la calidad de respuesta q (0 a 5).
// ponytail: standard SuperMemo SM-2; upgrade to FSRS if personalized forgetting curves requested.
func (uv *UserVocabulary) ApplySM2(q int) {
	if q < 0 {
		q = 0
	} else if q > 5 {
		q = 5
	}

	if uv.EaseFactor < 1.30 {
		uv.EaseFactor = 2.50
	}

	if q >= 3 {
		switch uv.RepetitionNumber {
		case 0:
			uv.IntervalDays = 1
		case 1:
			uv.IntervalDays = 6
		default:
			uv.IntervalDays = int(math.Round(float64(uv.IntervalDays) * uv.EaseFactor))
		}
		uv.RepetitionNumber++
		uv.TimesCorrect++
	} else {
		uv.RepetitionNumber = 0
		uv.IntervalDays = 1
	}

	diff := float64(5 - q)
	newEF := uv.EaseFactor + (0.1 - diff*(0.08+diff*0.02))
	if newEF < 1.30 {
		newEF = 1.30
	}
	uv.EaseFactor = math.Round(newEF*100) / 100

	uv.TimesSeen++
	now := time.Now()
	uv.LastReviewedAt = &now
	nextReview := now.Add(time.Duration(uv.IntervalDays) * 24 * time.Hour)
	uv.NextReviewDate = &nextReview
}

// UserLearningStats almacena estadísticas acumuladas (palabras, racha, precisión y estrellas).
type UserLearningStats struct {
	UserID            string     `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	WordsLearned      int        `gorm:"default:0;not null" json:"words_learned"`
	LessonsCompleted  int        `gorm:"default:0;not null" json:"lessons_completed"`
	GamesCompleted    int        `gorm:"default:0;not null" json:"games_completed"`
	TotalStars        int        `gorm:"default:0;not null" json:"total_stars"`
	TotalPoints       int        `gorm:"default:0;not null" json:"total_points"`
	CurrentStreakDays int        `gorm:"default:0;not null" json:"current_streak_days"`
	LongestStreakDays int        `gorm:"default:0;not null" json:"longest_streak_days"`
	AccuracyPercent   float64    `gorm:"type:decimal(5,2);default:0.00;not null" json:"accuracy_percent"`
	LastActivityDate  *time.Time `gorm:"type:date" json:"last_activity_date"`
	UpdatedAt         time.Time  `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// UserDailyActivity guarda el registro de estudio y preguntas respondidas por día.
type UserDailyActivity struct {
	ID                string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID            string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_activity,priority:1;index:idx_act_user_date,priority:1" json:"user_id"`
	ActivityDate      time.Time `gorm:"type:date;not null;uniqueIndex:idx_user_activity,priority:2;index:idx_act_user_date,priority:2" json:"activity_date"`
	MinutesStudied    int       `gorm:"default:0;not null" json:"minutes_studied"`
	QuestionsAnswered int       `gorm:"default:0;not null" json:"questions_answered"`
	CorrectAnswers    int       `gorm:"default:0;not null" json:"correct_answers"`
	PointsEarned      int       `gorm:"default:0;not null" json:"points_earned"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (a *UserDailyActivity) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
