package commands

import (
	"context"
	"errors"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type DeleteTeacherGameCommand struct {
	GameID    string
	TeacherID string
}

type DeleteTeacherGameHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewDeleteTeacherGameHandler(gameRepo repositories.TeacherGameRepository) *DeleteTeacherGameHandler {
	return &DeleteTeacherGameHandler{gameRepo: gameRepo}
}

// Handle elimina una actividad propia del docente. Si la actividad está vinculada
// a lecciones se archiva en lugar de borrarse, para no romper esos vínculos.
func (h *DeleteTeacherGameHandler) Handle(ctx context.Context, cmd DeleteTeacherGameCommand) error {
	if cmd.TeacherID == "" {
		return errors.New("el identificador del docente es obligatorio")
	}
	if cmd.GameID == "" {
		return errors.New("el identificador del juego es obligatorio")
	}

	game, err := h.gameRepo.FindByID(ctx, cmd.GameID)
	if err != nil {
		return err
	}
	if game == nil {
		return apperrors.New("No se encontró la actividad.")
	}
	if game.OwnerType == entities.OwnerTypePlatform {
		return apperrors.New("Solo puedes eliminar tus propias actividades.")
	}
	if game.TeacherID != cmd.TeacherID {
		return apperrors.New("No tienes permisos para eliminar esta actividad.")
	}

	usage, err := h.gameRepo.CountLessonUsage(ctx, cmd.GameID)
	if err != nil {
		return err
	}

	if usage > 0 {
		game.IsActive = false
		game.UpdatedAt = time.Now()
		return h.gameRepo.Update(ctx, game)
	}

	return h.gameRepo.Delete(ctx, cmd.GameID)
}
