package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLClassroomRepository implements ClassroomRepository using GORM and MySQL.
type MySQLClassroomRepository struct {
	db *gorm.DB
}

// NewMySQLClassroomRepository creates a MySQL repository instance for classrooms and cohorts.
func NewMySQLClassroomRepository(db *gorm.DB) domainRepo.ClassroomRepository {
	return &MySQLClassroomRepository{db: db}
}

// Create inserts a new virtual classroom into the database.
func (r *MySQLClassroomRepository) Create(ctx context.Context, classroom *entities.Classroom) error {
	return r.db.WithContext(ctx).Create(classroom).Error
}

// FindByID retrieves a classroom by its primary identifier with its relationships loaded.
func (r *MySQLClassroomRepository) FindByID(ctx context.Context, id string) (*entities.Classroom, error) {
	var classroom entities.Classroom
	err := r.db.WithContext(ctx).
		Preload("Teacher").
		Preload("Course").
		Preload("Enrollments.Student").
		Preload("Stumbles").
		First(&classroom, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &classroom, nil
}

// FindByCode locates a classroom by its unique alphanumeric code.
func (r *MySQLClassroomRepository) FindByCode(ctx context.Context, code string) (*entities.Classroom, error) {
	var classroom entities.Classroom
	err := r.db.WithContext(ctx).
		Preload("Teacher").
		Preload("Course").
		First(&classroom, "code = ?", code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &classroom, nil
}

// FindByTeacher lists all classrooms managed by a teacher with their cohorts and diagnostics.
func (r *MySQLClassroomRepository) FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error) {
	classrooms := make([]entities.Classroom, 0)
	err := r.db.WithContext(ctx).
		Where("teacher_id = ?", teacherID).
		Preload("Course").
		Preload("Enrollments.Student").
		Preload("Stumbles").
		Order("created_at DESC").
		Find(&classrooms).Error
	if err != nil {
		return nil, err
	}
	return classrooms, nil
}

// FindByStudent retrieves active classrooms where the student is enrolled, including progress and stumbles.
func (r *MySQLClassroomRepository) FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error) {
	classrooms := make([]entities.Classroom, 0)
	err := r.db.WithContext(ctx).
		Joins("JOIN classroom_enrollments ON classroom_enrollments.classroom_id = classrooms.id").
		Where("classroom_enrollments.student_id = ? AND classrooms.is_active = ?", studentID, true).
		Preload("Teacher").
		Preload("Course").
		Preload("Enrollments", "student_id = ?", studentID).
		Preload("Stumbles", func(db *gorm.DB) *gorm.DB {
			return db.Order("fail_rate DESC").Limit(10)
		}).
		Order("classrooms.created_at DESC").
		Find(&classrooms).Error
	if err != nil {
		return nil, err
	}
	if err := r.loadAssignedCourses(ctx, classrooms); err != nil {
		return nil, err
	}
	return classrooms, nil
}

// EnrollStudent registers a student in the virtual classroom.
func (r *MySQLClassroomRepository) EnrollStudent(ctx context.Context, enrollment *entities.ClassroomEnrollment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var classroom entities.Classroom
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", enrollment.ClassroomID).
			First(&classroom).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.New("El código ingresado no corresponde a un aula activa.")
			}
			return err
		}
		if !classroom.IsActive {
			return apperrors.New("El código ingresado no corresponde a un aula activa.")
		}

		var existing entities.ClassroomEnrollment
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("classroom_id = ? AND student_id = ?", enrollment.ClassroomID, enrollment.StudentID).
			First(&existing).Error
		if err == nil {
			*enrollment = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return tx.Create(enrollment).Error
	})
}

// FindEnrollment checks whether a student is already enrolled in a classroom.
func (r *MySQLClassroomRepository) FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error) {
	var enrollment entities.ClassroomEnrollment
	err := r.db.WithContext(ctx).
		First(&enrollment, "classroom_id = ? AND student_id = ?", classroomID, studentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &enrollment, nil
}

// ListEnrollments lists the students belonging to a cohort.
func (r *MySQLClassroomRepository) ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error) {
	enrollments := make([]entities.ClassroomEnrollment, 0)
	err := r.db.WithContext(ctx).
		Where("classroom_id = ?", classroomID).
		Preload("Student").
		Order("enrolled_at DESC").
		Find(&enrollments).Error
	if err != nil {
		return nil, err
	}
	return enrollments, nil
}

// ListStumbles retrieves words with a high failure rate in the classroom.
func (r *MySQLClassroomRepository) ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error) {
	stumbles := make([]entities.StudentStumble, 0)
	err := r.db.WithContext(ctx).
		Where("classroom_id = ?", classroomID).
		Order("fail_rate DESC").
		Find(&stumbles).Error
	if err != nil {
		return nil, err
	}
	return stumbles, nil
}

// CreateStumble inserts a pedagogical stumble record.
func (r *MySQLClassroomRepository) CreateStumble(ctx context.Context, stumble *entities.StudentStumble) error {
	return r.db.WithContext(ctx).Create(stumble).Error
}

// Update changes the attributes and configuration of an existing virtual classroom.
func (r *MySQLClassroomRepository) Update(ctx context.Context, classroom *entities.Classroom) error {
	return r.db.WithContext(ctx).Save(classroom).Error
}
