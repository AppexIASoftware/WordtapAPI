package commands

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
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
	if game.TeacherID != cmd.TeacherID {
		return apperrors.New("No tienes permisos para eliminar esta actividad.")
	}

	return h.gameRepo.Delete(ctx, cmd.GameID)
}
