package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLLessonGameRepository implements LessonGameRepository using GORM.
type MySQLLessonGameRepository struct {
	db *gorm.DB
}

// NewMySQLLessonGameRepository instantiates lesson-game repository.
func NewMySQLLessonGameRepository(db *gorm.DB) domainRepo.LessonGameRepository {
	return &MySQLLessonGameRepository{db: db}
}

// Create inserts a lesson-game link.
func (r *MySQLLessonGameRepository) Create(ctx context.Context, link *entities.LessonGame) error {
	return r.db.WithContext(ctx).Create(link).Error
}

// FindByLesson returns ordered lesson games for a lesson.
func (r *MySQLLessonGameRepository) FindByLesson(ctx context.Context, lessonID string) ([]entities.LessonGame, error) {
	links := make([]entities.LessonGame, 0)
	err := r.db.WithContext(ctx).
		Preload("TeacherGame").
		Where("lesson_id = ?", lessonID).
		Order("sort_order ASC, created_at ASC").
		Find(&links).Error
	if err != nil {
		return nil, err
	}
	return links, nil
}

// FindByLessonAndGame finds a single lesson-game link.
func (r *MySQLLessonGameRepository) FindByLessonAndGame(ctx context.Context, lessonID, gameID string) (*entities.LessonGame, error) {
	var link entities.LessonGame
	err := r.db.WithContext(ctx).
		Preload("TeacherGame").
		Where("lesson_id = ? AND teacher_game_id = ?", lessonID, gameID).
		First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &link, nil
}

// Update persists sort order and required flag with explicit columns.
func (r *MySQLLessonGameRepository) Update(ctx context.Context, link *entities.LessonGame) error {
	return r.db.WithContext(ctx).
		Model(&entities.LessonGame{}).
		Where("id = ?", link.ID).
		Updates(map[string]any{
			"sort_order":  link.SortOrder,
			"is_required": link.IsRequired,
		}).Error
}

// Delete removes link idempotently.
func (r *MySQLLessonGameRepository) Delete(ctx context.Context, lessonID, gameID string) error {
	return r.db.WithContext(ctx).
		Where("lesson_id = ? AND teacher_game_id = ?", lessonID, gameID).
		Delete(&entities.LessonGame{}).Error
}

// NextSortOrder returns next available sort order in lesson.
func (r *MySQLLessonGameRepository) NextSortOrder(ctx context.Context, lessonID string) (int, error) {
	var next int
	row := r.db.WithContext(ctx).
		Model(&entities.LessonGame{}).
		Where("lesson_id = ?", lessonID).
		Select("COALESCE(MAX(sort_order), -1) + 1").
		Row()
	if err := row.Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

// CountByGameIDs counts lesson links grouped by game ID.
func (r *MySQLLessonGameRepository) CountByGameIDs(ctx context.Context, ids []string) (map[string]int, error) {
	return countLessonGamesByGameIDs(ctx, r.db, ids)
}
