package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// LessonRepository define el contrato de persistencia para lecciones y contenido didáctico.
type LessonRepository interface {
	FindByIDOrSlug(ctx context.Context, identifier string) (*entities.Lesson, error)
}
