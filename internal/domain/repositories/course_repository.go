package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// CourseRepository defines persistence operations for course management.
type CourseRepository interface {
	Create(ctx context.Context, course *entities.Course) error
	FindByAuthor(ctx context.Context, authorID string) ([]entities.Course, error)
	FindPublished(ctx context.Context) ([]entities.Course, error)
	FindByID(ctx context.Context, id string) (*entities.Course, error)
	Update(ctx context.Context, course *entities.Course) error
}
