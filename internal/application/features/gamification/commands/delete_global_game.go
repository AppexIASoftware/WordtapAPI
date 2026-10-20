package commands

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type DeleteGlobalGameCommand struct {
	GameID string
}

type DeleteGlobalGameHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewDeleteGlobalGameHandler(gameRepo repositories.TeacherGameRepository) *DeleteGlobalGameHandler {
	return &DeleteGlobalGameHandler{gameRepo: gameRepo}
}

// Handle deletes a platform-global activity. Authorization is the admin/moderator
// role gate at the route, so ownership is never compared here; instead, this
// handler refuses any activity that is not global, which stops an administrator
// from deleting a teacher's own activity through the admin endpoints.
func (h *DeleteGlobalGameHandler) Handle(ctx context.Context, cmd DeleteGlobalGameCommand) error {
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
	if !game.IsGlobal {
		return apperrors.New("Solo se pueden gestionar actividades de la plataforma.")
	}

	return h.gameRepo.Delete(ctx, cmd.GameID)
}
