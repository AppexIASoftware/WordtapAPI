package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"gorm.io/gorm"
)

// GameLessonLinks loads every link, including incomplete relations, to fail closed.
type GameLessonLinks struct{ db *gorm.DB }

func NewGameLessonLinks(db *gorm.DB) *GameLessonLinks { return &GameLessonLinks{db: db} }

func (r *GameLessonLinks) FindLessonsByGame(ctx context.Context, gameID string) ([]entities.LessonGame, error) {
	var links []entities.LessonGame
	err := r.db.WithContext(ctx).Preload("Lesson.Course").Where("teacher_game_id = ?", gameID).Find(&links).Error
	return links, err
}
