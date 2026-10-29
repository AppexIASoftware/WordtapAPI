package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// ClassroomRepository define las operaciones de persistencia para aulas virtuales, cohortes y diagnósticos.
type ClassroomRepository interface {
	Create(ctx context.Context, classroom *entities.Classroom) error
	FindByID(ctx context.Context, id string) (*entities.Classroom, error)
	FindByCode(ctx context.Context, code string) (*entities.Classroom, error)
	FindByTeacher(ctx context.Context, teacherID string) ([]entities.Classroom, error)
	FindByStudent(ctx context.Context, studentID string) ([]entities.Classroom, error)
	EnrollStudent(ctx context.Context, enrollment *entities.ClassroomEnrollment) error
	FindEnrollment(ctx context.Context, classroomID, studentID string) (*entities.ClassroomEnrollment, error)
	ListEnrollments(ctx context.Context, classroomID string) ([]entities.ClassroomEnrollment, error)
	ListStumbles(ctx context.Context, classroomID string) ([]entities.StudentStumble, error)
	CreateStumble(ctx context.Context, stumble *entities.StudentStumble) error
	Update(ctx context.Context, classroom *entities.Classroom) error
}
