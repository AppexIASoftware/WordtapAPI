package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// LessonRepository define el contrato de persistencia para lecciones y contenido didáctico.
type LessonRepository interface {
	FindByIDOrSlug(ctx context.Context, identifier string) (*entities.Lesson, error)
	FindByCourseID(ctx context.Context, courseID string) ([]entities.Lesson, error)
	FindByID(ctx context.Context, id string) (*entities.Lesson, error)
	Create(ctx context.Context, lesson *entities.Lesson) error
	Update(ctx context.Context, lesson *entities.Lesson) error
	Delete(ctx context.Context, id string) error
	SaveItem(ctx context.Context, item *entities.LessonItem) error
	DeleteItem(ctx context.Context, itemID string) error
}
