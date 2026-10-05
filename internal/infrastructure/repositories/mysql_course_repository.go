package repositories

import (
	"context"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLCourseRepository implements CourseRepository using GORM and MySQL.
type MySQLCourseRepository struct {
	db *gorm.DB
}

// NewMySQLCourseRepository creates a new MySQL repository instance for courses.
func NewMySQLCourseRepository(db *gorm.DB) domainRepo.CourseRepository {
	return &MySQLCourseRepository{db: db}
}

// Create inserts a new course into the database.
func (r *MySQLCourseRepository) Create(ctx context.Context, course *entities.Course) error {
	return r.db.WithContext(ctx).Create(course).Error
}

// FindByAuthor returns all courses authored by a specific user.
func (r *MySQLCourseRepository) FindByAuthor(ctx context.Context, authorID string) ([]entities.Course, error) {
	courses := make([]entities.Course, 0)
	err := r.db.WithContext(ctx).
		Where("created_by = ?", authorID).
		Preload("Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("lessons.sort_order ASC, lessons.created_at ASC")
		}).
		Preload("Lessons.Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_items.sort_order ASC")
		}).
		Preload("Lessons.Items.VocabularyItem").
		Preload("Lessons.Items.Phrase").
		Order("created_at DESC").
		Find(&courses).Error
	if err != nil {
		return nil, err
	}
	return courses, nil
}

// FindPublished returns all published courses available for the public catalog.
func (r *MySQLCourseRepository) FindPublished(ctx context.Context) ([]entities.Course, error) {
	courses := make([]entities.Course, 0)
	err := r.db.WithContext(ctx).
		Where("status = ?", entities.ContentStatusPublished).
		Preload("Lessons").
		Order("sort_order ASC, created_at DESC").
		Find(&courses).Error
	if err != nil {
		return nil, err
	}
	return courses, nil
}

// FindByID retrieves a single course by its ID.
func (r *MySQLCourseRepository) FindByID(ctx context.Context, id string) (*entities.Course, error) {
	var course entities.Course
	err := r.db.WithContext(ctx).
		Preload("Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("lessons.sort_order ASC, lessons.created_at ASC")
		}).
		Preload("Lessons.Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_items.sort_order ASC")
		}).
		Preload("Lessons.Items.VocabularyItem").
		Preload("Lessons.Items.Phrase").
		Where("id = ?", id).
		First(&course).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

// Update updates an existing course entity.
func (r *MySQLCourseRepository) Update(ctx context.Context, course *entities.Course) error {
	return r.db.WithContext(ctx).Save(course).Error
}

// Delete removes a course by its ID.
func (r *MySQLCourseRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entities.Course{}).Error
}
