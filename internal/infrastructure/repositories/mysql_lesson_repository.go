package repositories

import (
	"context"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLLessonRepository implementa LessonRepository utilizando GORM y MySQL.
type MySQLLessonRepository struct {
	db *gorm.DB
}

// NewMySQLLessonRepository crea una nueva instancia del repositorio MySQL para lecciones.
func NewMySQLLessonRepository(db *gorm.DB) domainRepo.LessonRepository {
	return &MySQLLessonRepository{db: db}
}

// FindByIDOrSlug busca una lección por su ID o slug, precargando su categoría, curso e ítems ordenados.
func (r *MySQLLessonRepository) FindByIDOrSlug(ctx context.Context, identifier string) (*entities.Lesson, error) {
	var lesson entities.Lesson
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Course").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_items.sort_order ASC")
		}).
		Preload("Items.VocabularyItem").
		Preload("Items.Phrase").
		Where("id = ? OR slug = ?", identifier, identifier).
		First(&lesson).Error

	if err != nil {
		return nil, err
	}
	return &lesson, nil
}
