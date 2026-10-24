package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// UserRepository define el contrato de persistencia para usuarios, cuentas OAuth y sesiones.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entities.User, error)
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	FindAuthAccount(ctx context.Context, provider, providerAccountID string) (*entities.UserAuthAccount, error)
	CreateWithInitialState(ctx context.Context, user *entities.User, authAccount *entities.UserAuthAccount) error
	CreateAuthAccount(ctx context.Context, authAccount *entities.UserAuthAccount) error
	CreateSession(ctx context.Context, session *entities.UserSession) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (*entities.UserSession, error)
	RotateSession(ctx context.Context, oldTokenHash string, newSession *entities.UserSession) (bool, error)
	RevokeSession(ctx context.Context, tokenHash string) error
	RevokeAllSessions(ctx context.Context, userID string) error
	UpdateUser(ctx context.Context, user *entities.User) error
}

// RecordProgressParams contiene los parámetros para registrar la finalización exitosa de un juego.
type RecordProgressParams struct {
	UserID         string
	LessonID       string
	StarsEarned    int
	WordsPracticed int
}

// RecordProgressResult contiene el estado actualizado de progreso tras completar un juego.
type RecordProgressResult struct {
	LessonCompleted   bool `json:"lesson_completed"`
	StarsEarned       int  `json:"stars_earned"`
	TotalStars        int  `json:"total_stars"`
	TotalPoints       int  `json:"total_points"`
	CurrentStreakDays int  `json:"current_streak_days"`
	WordsLearned      int  `json:"words_learned"`
	LessonsCompleted  int  `json:"lessons_completed"`
	GamesCompleted    int  `json:"games_completed"`
}

// UserLearningStatsRepository define las operaciones de persistencia para telemetría de aprendizaje, vidas y rachas.
type UserLearningStatsRepository interface {
	GetLearningStats(ctx context.Context, userID string) (*entities.UserLearningStats, error)
	DeductUserLife(ctx context.Context, userID string) (*entities.UserLearningStats, error)
	RefillUserLives(ctx context.Context, userID string) (*entities.UserLearningStats, error)
	RecordGameVictory(ctx context.Context, params RecordProgressParams) (*RecordProgressResult, error)
	GetCompletedLessonIDs(ctx context.Context, userID string) ([]string, error)
}
