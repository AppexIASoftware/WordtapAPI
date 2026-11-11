package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLTeacherGameRepository implements TeacherGameRepository using GORM and MySQL.
type MySQLTeacherGameRepository struct {
	db *gorm.DB
}

// NewMySQLTeacherGameRepository creates the repository for teacher games.
func NewMySQLTeacherGameRepository(db *gorm.DB) domainRepo.TeacherGameRepository {
	return &MySQLTeacherGameRepository{db: db}
}

// Create inserts a custom game into the database.
func (r *MySQLTeacherGameRepository) Create(ctx context.Context, game *entities.TeacherGame) error {
	return r.db.WithContext(ctx).Create(game).Error
}

// FindByID retrieves a game by ID with its associated course.
func (r *MySQLTeacherGameRepository) FindByID(ctx context.Context, id string) (*entities.TeacherGame, error) {
	var game entities.TeacherGame
	err := r.db.WithContext(ctx).
		Preload("Course").
		Preload("Category").
		First(&game, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &game, nil
}

// FindByTeacher lists games created by a teacher in chronological order.
func (r *MySQLTeacherGameRepository) FindByTeacher(ctx context.Context, teacherID string) ([]entities.TeacherGame, error) {
	games := make([]entities.TeacherGame, 0)
	err := r.db.WithContext(ctx).
		Where("teacher_id = ?", teacherID).
		Preload("Course").
		Preload("Category").
		Order("created_at DESC").
		Find(&games).Error
	if err != nil {
		return nil, err
	}
	if err := r.fillUsage(ctx, games); err != nil {
		return nil, err
	}
	return games, nil
}

// FindGlobal excludes ambiguous legacy records that identify a specific lesson.
func (r *MySQLTeacherGameRepository) FindGlobal(ctx context.Context) ([]entities.TeacherGame, error) {
	games := make([]entities.TeacherGame, 0)
	err := r.db.WithContext(ctx).
		Where("is_global = ? AND is_active = ?", true, true).
		Where("NOT (source_type = ? AND source_id IS NOT NULL AND source_id <> '')", "course_lesson").
		Preload("Teacher").
		Preload("Category").
		Order(accessTierOrder("access_tier") + ", created_at DESC").
		Find(&games).Error
	if err != nil {
		return nil, err
	}
	if err := r.fillUsage(ctx, games); err != nil {
		return nil, err
	}
	return games, nil
}

// FindStandaloneByTeachers retrieves standalone published games created by the specified teachers.
func (r *MySQLTeacherGameRepository) FindStandaloneByTeachers(ctx context.Context, teacherIDs []string) ([]entities.TeacherGame, error) {
	if len(teacherIDs) == 0 {
		return []entities.TeacherGame{}, nil
	}
	games := make([]entities.TeacherGame, 0)
	err := r.db.WithContext(ctx).
		Where("teacher_id IN ? AND is_active = ? AND is_published_standalone = ?", teacherIDs, true, true).
		Preload("Teacher").
		Preload("Course").
		Preload("Category").
		Order("created_at DESC").
		Find(&games).Error
	if err != nil {
		return nil, err
	}
	return games, nil
}

// FindAll returns every teacher game regardless of author (for admin management).
func (r *MySQLTeacherGameRepository) FindAll(ctx context.Context) ([]entities.TeacherGame, error) {
	games := make([]entities.TeacherGame, 0)
	err := r.db.WithContext(ctx).
		Preload("Teacher").
		Preload("Course").
		Preload("Category").
		Order("created_at DESC").
		Find(&games).Error
	if err != nil {
		return nil, err
	}
	if err := r.fillUsage(ctx, games); err != nil {
		return nil, err
	}
	return games, nil
}

// fillUsage populates UsageCount with the grouped count of lesson links.
func (r *MySQLTeacherGameRepository) fillUsage(ctx context.Context, games []entities.TeacherGame) error {
	ids := make([]string, 0, len(games))
	for _, game := range games {
		ids = append(ids, game.ID)
	}

	usage, err := countLessonGamesByGameIDs(ctx, r.db, ids)
	if err != nil {
		return err
	}

	for i := range games {
		games[i].UsageCount = usage[games[i].ID]
	}
	return nil
}

// Update persists only the mutable game columns. The explicit list prevents
// rewriting associations and fields that no handler modifies.
func (r *MySQLTeacherGameRepository) Update(ctx context.Context, game *entities.TeacherGame) error {
	return r.db.WithContext(ctx).
		Model(&entities.TeacherGame{}).
		Where("id = ?", game.ID).
		Updates(map[string]any{
			"base_archetype":          game.BaseArchetype,
			"course_id":               game.CourseID,
			"category_id":             game.CategoryID,
			"title":                   game.Title,
			"description":             game.Description,
			"level":                   game.Level,
			"access_tier":             game.AccessTier,
			"time_limit_seconds":      game.TimeLimitSeconds,
			"allowed_mistakes":        game.AllowedMistakes,
			"reward_xp":               game.RewardXP,
			"item_count":              game.ItemCount,
			"source_type":             game.SourceType,
			"source_id":               game.SourceID,
			"custom_words_json":       game.CustomWordsJSON,
			"is_published_standalone": game.IsPublishedStandalone,
			"audience_scope":          game.AudienceScope,
			"assigned_course_ids":     game.AssignedCourseIDs,
			"assigned_classroom_ids":  game.AssignedClassroomIDs,
			"is_active":               game.IsActive,
			"updated_at":              game.UpdatedAt,
		}).Error
}

// Delete permanently removes a teacher's custom game and any lesson links it has.
func (r *MySQLTeacherGameRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("teacher_game_id = ?", id).Delete(&entities.LessonGame{}).Error; err != nil {
			return err
		}
		return tx.Delete(&entities.TeacherGame{}, "id = ?", id).Error
	})
}

// CountLessonUsage counts how many lessons use an activity.
func (r *MySQLTeacherGameRepository) CountLessonUsage(ctx context.Context, gameID string) (int, error) {
	return countLessonGamesByGameID(ctx, r.db, gameID)
}
