package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLClassroomRepository implementa ClassroomRepository mediante GORM y MySQL.
type MySQLClassroomRepository struct {
	db *gorm.DB
}

// NewMySQLClassroomRepository crea una instancia del repositorio MySQL para aulas y cohortes.
func NewMySQLClassroomRepository(db *gorm.DB) domainRepo.ClassroomRepository {
	return &MySQLClassroomRepository{db: db}
}

// Create inserta una nueva aula virtual en la base de datos.
func (r *MySQLClassroomRepository) Create(ctx context.Context, classroom *entities.Classroom) error {
	return r.db.WithContext(ctx).Create(classroom).Error
}

// FindByID recupera un aula por su identificador primario con sus relaciones cargadas.
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

// FindByCode localiza un aula por su código alfanumérico único.
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

// FindByTeacher lista todas las aulas administradas por un docente con sus cohortes y diagnósticos.
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

// EnrollStudent registra a un alumno en el aula virtual.
func (r *MySQLClassroomRepository) EnrollStudent(ctx context.Context, enrollment *entities.ClassroomEnrollment) error {
	return r.db.WithContext(ctx).Create(enrollment).Error
}

// FindEnrollment comprueba si un estudiante ya está inscrito en un aula.
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

// ListEnrollments lista los estudiantes pertenecientes a una cohorte.
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

// ListStumbles consulta las palabras de alta tasa de fallo en el aula.
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

// CreateStumble inserta un registro de tropiezo pedagógico.
func (r *MySQLClassroomRepository) CreateStumble(ctx context.Context, stumble *entities.StudentStumble) error {
	return r.db.WithContext(ctx).Create(stumble).Error
}

// Update actualiza los atributos y configuración de un aula virtual existente.
func (r *MySQLClassroomRepository) Update(ctx context.Context, classroom *entities.Classroom) error {
	return r.db.WithContext(ctx).Save(classroom).Error
}
