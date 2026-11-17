package repositories

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLLessonRepository implements LessonRepository using GORM.
type MySQLLessonRepository struct {
	db *gorm.DB
}

// NewMySQLLessonRepository instantiates MySQL lesson repository.
func NewMySQLLessonRepository(db *gorm.DB) domainRepo.LessonRepository {
	return &MySQLLessonRepository{db: db}
}

// FindByIDOrSlug finds lesson by ID or slug with preloaded cards.
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
		Preload("Items.ContentBankItem").
		Where("id = ? OR slug = ?", identifier, identifier).
		First(&lesson).Error

	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

// FindByCourseID returns lessons for a course ordered by sort order.
func (r *MySQLLessonRepository) FindByCourseID(ctx context.Context, courseID string) ([]entities.Lesson, error) {
	lessons := make([]entities.Lesson, 0)
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_items.sort_order ASC")
		}).
		Preload("Items.VocabularyItem").
		Preload("Items.Phrase").
		Preload("Items.ContentBankItem").
		Order(accessTierOrder("access_tier") + ", sort_order ASC, created_at ASC").
		Find(&lessons).Error
	if err != nil {
		return nil, err
	}
	return lessons, nil
}

// FindByID finds lesson by ID with preloaded cards.
func (r *MySQLLessonRepository) FindByID(ctx context.Context, id string) (*entities.Lesson, error) {
	var lesson entities.Lesson
	err := r.db.WithContext(ctx).
		Preload("Course").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_items.sort_order ASC")
		}).
		Preload("Items.VocabularyItem").
		Preload("Items.Phrase").
		Preload("Items.ContentBankItem").
		Where("id = ?", id).
		First(&lesson).Error
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

// Create inserts a new lesson.
func (r *MySQLLessonRepository) Create(ctx context.Context, lesson *entities.Lesson) error {
	return r.db.WithContext(ctx).Create(lesson).Error
}

// Update updates an existing lesson.
func (r *MySQLLessonRepository) Update(ctx context.Context, lesson *entities.Lesson) error {
	return r.db.WithContext(ctx).Save(lesson).Error
}

// Delete removes a lesson by ID.
func (r *MySQLLessonRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entities.Lesson{}).Error
}

// SaveItem saves a lesson item while rejecting cross-lesson overwrites by ID.
func (r *MySQLLessonRepository) SaveItem(ctx context.Context, item *entities.LessonItem) error {
	if item == nil {
		return apperrors.New("Tarjeta inválida.")
	}
	if strings.TrimSpace(item.ID) != "" && strings.TrimSpace(item.LessonID) != "" {
		var existing entities.LessonItem
		err := r.db.WithContext(ctx).Select("id, lesson_id").Where("id = ?", item.ID).Take(&existing).Error
		if err == nil && existing.LessonID != item.LessonID {
			return apperrors.New("La tarjeta pertenece a otra lección.")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return r.db.WithContext(ctx).Save(item).Error
}

// DeleteItem removes a lesson item scoped to its owning lesson.
func (r *MySQLLessonRepository) DeleteItem(ctx context.Context, lessonID, itemID string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND lesson_id = ?", itemID, lessonID).
		Delete(&entities.LessonItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.New("No se encontró la tarjeta en esta lección.")
	}
	return nil
}
