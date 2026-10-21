package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLTeacherGameRepository implementa TeacherGameRepository mediante GORM y MySQL.
type MySQLTeacherGameRepository struct {
	db *gorm.DB
}

// NewMySQLTeacherGameRepository instancia el repositorio para juegos del docente.
func NewMySQLTeacherGameRepository(db *gorm.DB) domainRepo.TeacherGameRepository {
	if db != nil {
		_ = db.AutoMigrate(&entities.TeacherGame{})
	}
	return &MySQLTeacherGameRepository{db: db}
}

// Create inserta un juego personalizado en la base de datos.
func (r *MySQLTeacherGameRepository) Create(ctx context.Context, game *entities.TeacherGame) error {
	return r.db.WithContext(ctx).Create(game).Error
}

// FindByID recupera un juego por ID con su curso asociado.
func (r *MySQLTeacherGameRepository) FindByID(ctx context.Context, id string) (*entities.TeacherGame, error) {
	var game entities.TeacherGame
	err := r.db.WithContext(ctx).
		Preload("Course").
		First(&game, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &game, nil
}

// FindByTeacher lista los juegos creados por un docente ordenados cronológicamente.
func (r *MySQLTeacherGameRepository) FindByTeacher(ctx context.Context, teacherID string) ([]entities.TeacherGame, error) {
	games := make([]entities.TeacherGame, 0)
	err := r.db.WithContext(ctx).
		Where("teacher_id = ?", teacherID).
		Preload("Course").
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

// FindGlobal lista las actividades de plataforma disponibles para cualquier curso.
// owner_type es la única autoridad de propiedad; is_global queda como legado.
func (r *MySQLTeacherGameRepository) FindGlobal(ctx context.Context) ([]entities.TeacherGame, error) {
	games := make([]entities.TeacherGame, 0)
	err := r.db.WithContext(ctx).
		Where("owner_type = ? AND is_active = ?", entities.OwnerTypePlatform, true).
		Preload("Teacher").
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

// FindAll returns every teacher game regardless of author (for admin management).
func (r *MySQLTeacherGameRepository) FindAll(ctx context.Context) ([]entities.TeacherGame, error) {
	games := make([]entities.TeacherGame, 0)
	err := r.db.WithContext(ctx).
		Preload("Teacher").
		Preload("Course").
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

// fillUsage completa UsageCount con el conteo agrupado de vínculos en lecciones.
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

// Update persiste únicamente las columnas mutables del juego. La lista explícita
// evita reescribir asociaciones y campos que ningún handler modifica.
func (r *MySQLTeacherGameRepository) Update(ctx context.Context, game *entities.TeacherGame) error {
	return r.db.WithContext(ctx).
		Model(&entities.TeacherGame{}).
		Where("id = ?", game.ID).
		Updates(map[string]any{
			"course_id":          game.CourseID,
			"title":              game.Title,
			"description":        game.Description,
			"level":              game.Level,
			"time_limit_seconds": game.TimeLimitSeconds,
			"allowed_mistakes":   game.AllowedMistakes,
			"reward_xp":          game.RewardXP,
			"item_count":         game.ItemCount,
			"source_type":        game.SourceType,
			"source_id":          game.SourceID,
			"custom_words_json":  game.CustomWordsJSON,
			"is_active":          game.IsActive,
			"updated_at":         game.UpdatedAt,
		}).Error
}

// Delete elimina físicamente un juego personalizado del docente.
func (r *MySQLTeacherGameRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entities.TeacherGame{}, "id = ?", id).Error
}

// CountLessonUsage cuenta cuántas lecciones usan una actividad.
func (r *MySQLTeacherGameRepository) CountLessonUsage(ctx context.Context, gameID string) (int, error) {
	return countLessonGamesByGameID(ctx, r.db, gameID)
}
