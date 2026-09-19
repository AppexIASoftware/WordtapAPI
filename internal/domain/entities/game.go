package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GameType define los 12 tipos de minijuegos disponibles en WordTap.
type GameType string

const (
	GameTypeMatch            GameType = "match"
	GameTypeMemory           GameType = "memory"
	GameTypeWordSearch       GameType = "word_search"
	GameTypeHangman          GameType = "hangman"
	GameTypeSpeedTranslation GameType = "speed_translation"
	GameTypeMultipleChoice   GameType = "multiple_choice"
	GameTypeWordScramble     GameType = "word_scramble"
	GameTypeSentenceBuilder  GameType = "sentence_builder"
	GameTypeFillIn           GameType = "fill_in"
	GameTypeCrossword        GameType = "crossword"
	GameTypeListening        GameType = "listening"
	GameTypePhrasalVerbs     GameType = "phrasal_verbs"
)

// Game define una actividad interactiva en la ruta de juegos con estrellas para desbloqueo.
type Game struct {
	ID               string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	LessonID         *string       `gorm:"type:varchar(36);index" json:"lesson_id"`
	Title            string        `gorm:"type:varchar(180);not null" json:"title"`
	Description      *string       `gorm:"type:text" json:"description"`
	GameType         GameType      `gorm:"type:varchar(40);not null" json:"game_type"`
	AccessTier       AccessTier    `gorm:"type:varchar(20);default:'free';not null" json:"access_tier"`
	Status           ContentStatus `gorm:"type:varchar(20);default:'draft';not null" json:"status"`
	Level            int           `gorm:"default:1;not null" json:"level"`
	SortOrder        int           `gorm:"default:0;not null" json:"sort_order"`
	UnlockStars      int           `gorm:"default:0;not null" json:"unlock_stars"`
	TimeLimitSeconds *int          `json:"time_limit_seconds"`
	CreatedAt        time.Time     `json:"created_at"`

	Lesson    *Lesson        `gorm:"foreignKey:LessonID;constraint:OnDelete:SET NULL" json:"lesson,omitempty"`
	Questions []GameQuestion `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE" json:"questions,omitempty"`
}

func (g *Game) BeforeCreate(tx *gorm.DB) error {
	if g.ID == "" {
		g.ID = uuid.NewString()
	}
	return nil
}

// GameQuestion almacena preguntas, pares de fichas y datos dinámicos (JSON) de cada minijuego.
type GameQuestion struct {
	ID               string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID           string    `gorm:"type:varchar(36);not null;index:idx_questions_game,priority:1" json:"game_id"`
	VocabularyItemID *string   `gorm:"type:varchar(36);index" json:"vocabulary_item_id"`
	PhraseID         *string   `gorm:"type:varchar(36);index" json:"phrase_id"`
	Prompt           string    `gorm:"type:text;not null" json:"prompt"`
	QuestionData     string    `gorm:"type:json;not null" json:"question_data"`
	CorrectAnswer    string    `gorm:"type:text;not null" json:"correct_answer"`
	Explanation      *string   `gorm:"type:text" json:"explanation"`
	SortOrder        int       `gorm:"default:0;not null;index:idx_questions_game,priority:2" json:"sort_order"`
	Points           int       `gorm:"default:10;not null" json:"points"`
	CreatedAt        time.Time `json:"created_at"`

	Game           *Game           `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE" json:"game,omitempty"`
	VocabularyItem *VocabularyItem `gorm:"foreignKey:VocabularyItemID;constraint:OnDelete:SET NULL" json:"vocabulary_item,omitempty"`
	Phrase         *Phrase         `gorm:"foreignKey:PhraseID;constraint:OnDelete:SET NULL" json:"phrase,omitempty"`
}

func (q *GameQuestion) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.NewString()
	}
	if q.QuestionData == "" {
		q.QuestionData = "{}"
	}
	return nil
}

// GameAttempt registra el resultado, estrellas (0-3) y precisión de una partida jugada.
type GameAttempt struct {
	ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID          string    `gorm:"type:varchar(36);not null;index:idx_attempts_user,priority:1" json:"user_id"`
	GameID          string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	Score           int       `gorm:"default:0;not null" json:"score"`
	Stars           int       `gorm:"default:0;not null" json:"stars"`
	TotalQuestions  int       `gorm:"default:0;not null" json:"total_questions"`
	CorrectAnswers  int       `gorm:"default:0;not null" json:"correct_answers"`
	AccuracyPercent float64   `gorm:"type:decimal(5,2);default:0.00;not null" json:"accuracy_percent"`
	DurationSeconds *int      `json:"duration_seconds"`
	CompletedAt     time.Time `gorm:"index:idx_attempts_user,priority:2" json:"completed_at"`

	User    *User        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Game    *Game        `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE" json:"game,omitempty"`
	Answers []GameAnswer `gorm:"foreignKey:AttemptID;constraint:OnDelete:CASCADE" json:"answers,omitempty"`
}

func (a *GameAttempt) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.TotalQuestions > 0 {
		a.AccuracyPercent = (float64(a.CorrectAnswers) / float64(a.TotalQuestions)) * 100.0
	}
	return nil
}

// GameAnswer registra las respuestas enviadas por pregunta para análisis de errores.
type GameAnswer struct {
	ID             string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	AttemptID      string    `gorm:"type:varchar(36);not null;index" json:"attempt_id"`
	QuestionID     string    `gorm:"type:varchar(36);not null;index" json:"question_id"`
	AnswerText     *string   `gorm:"type:text" json:"answer_text"`
	IsCorrect      bool      `gorm:"not null" json:"is_correct"`
	ResponseTimeMS *int      `json:"response_time_ms"`
	CreatedAt      time.Time `json:"created_at"`

	Attempt  *GameAttempt  `gorm:"foreignKey:AttemptID;constraint:OnDelete:CASCADE" json:"attempt,omitempty"`
	Question *GameQuestion `gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE" json:"question,omitempty"`
}

func (ga *GameAnswer) BeforeCreate(tx *gorm.DB) error {
	if ga.ID == "" {
		ga.ID = uuid.NewString()
	}
	return nil
}

// UserGameUnlock permite desbloquear juegos de forma no lineal para el usuario.
type UserGameUnlock struct {
	UserID     string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	GameID     string    `gorm:"type:varchar(36);primaryKey" json:"game_id"`
	UnlockedAt time.Time `json:"unlocked_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Game *Game `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE" json:"game,omitempty"`
}
