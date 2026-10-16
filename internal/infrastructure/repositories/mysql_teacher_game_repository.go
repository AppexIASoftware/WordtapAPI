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
	return games, nil
}

// Update persiste cambios sobre un juego personalizado.
func (r *MySQLTeacherGameRepository) Update(ctx context.Context, game *entities.TeacherGame) error {
	return r.db.WithContext(ctx).Save(game).Error
}

// Delete elimina físicamente un juego personalizado del docente.
func (r *MySQLTeacherGameRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entities.TeacherGame{}, "id = ?", id).Error
}
