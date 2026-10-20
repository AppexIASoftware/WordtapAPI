package repositories

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// TeacherGameRepository define las operaciones de persistencia para actividades y minijuegos personalizados del docente.
type TeacherGameRepository interface {
	Create(ctx context.Context, game *entities.TeacherGame) error
	FindByID(ctx context.Context, id string) (*entities.TeacherGame, error)
	FindByTeacher(ctx context.Context, teacherID string) ([]entities.TeacherGame, error)
	FindAll(ctx context.Context) ([]entities.TeacherGame, error)
	Update(ctx context.Context, game *entities.TeacherGame) error
	Delete(ctx context.Context, id string) error
}
