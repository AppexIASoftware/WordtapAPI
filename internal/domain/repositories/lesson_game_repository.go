package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// LessonGameRepository defines persistence operations for lesson-game links.
type LessonGameRepository interface {
	Create(ctx context.Context, link *entities.LessonGame) error
	FindByLesson(ctx context.Context, lessonID string) ([]entities.LessonGame, error)
	FindByLessonAndGame(ctx context.Context, lessonID, gameID string) (*entities.LessonGame, error)
	Update(ctx context.Context, link *entities.LessonGame) error
	Delete(ctx context.Context, lessonID, gameID string) error
	NextSortOrder(ctx context.Context, lessonID string) (int, error)
	CountByGameIDs(ctx context.Context, ids []string) (map[string]int, error)
}
