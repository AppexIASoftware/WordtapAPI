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
		Where("id = ? OR slug = ?", identifier, identifier).
		First(&lesson).Error

	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

// FindByCourseID devuelve todas las lecciones asociadas a un curso específico ordenadas por sort_order.
func (r *MySQLLessonRepository) FindByCourseID(ctx context.Context, courseID string) ([]entities.Lesson, error) {
	lessons := make([]entities.Lesson, 0)
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_items.sort_order ASC")
		}).
		Order("sort_order ASC, created_at ASC").
		Find(&lessons).Error
	if err != nil {
		return nil, err
	}
	return lessons, nil
}

// FindByID busca una lección por su ID primario.
func (r *MySQLLessonRepository) FindByID(ctx context.Context, id string) (*entities.Lesson, error) {
	var lesson entities.Lesson
	err := r.db.WithContext(ctx).
		Preload("Course").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_items.sort_order ASC")
		}).
		Where("id = ?", id).
		First(&lesson).Error
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

// Create inserta una nueva lección en la base de datos.
func (r *MySQLLessonRepository) Create(ctx context.Context, lesson *entities.Lesson) error {
	return r.db.WithContext(ctx).Create(lesson).Error
}

// Update actualiza los campos de una lección existente.
func (r *MySQLLessonRepository) Update(ctx context.Context, lesson *entities.Lesson) error {
	return r.db.WithContext(ctx).Save(lesson).Error
}

// Delete elimina una lección por su ID.
func (r *MySQLLessonRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entities.Lesson{}).Error
}

// SaveItem inserta o actualiza una tarjeta/paso dentro de una lección.
func (r *MySQLLessonRepository) SaveItem(ctx context.Context, item *entities.LessonItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// DeleteItem elimina un ítem de lección por su ID.
func (r *MySQLLessonRepository) DeleteItem(ctx context.Context, itemID string) error {
	return r.db.WithContext(ctx).Where("id = ?", itemID).Delete(&entities.LessonItem{}).Error
}
