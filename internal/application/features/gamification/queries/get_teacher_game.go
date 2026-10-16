package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type GetTeacherGameDetailQuery struct {
	GameID    string
	TeacherID string
}

type GetTeacherGameDetailHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewGetTeacherGameDetailHandler(gameRepo repositories.TeacherGameRepository) *GetTeacherGameDetailHandler {
	return &GetTeacherGameDetailHandler{gameRepo: gameRepo}
}

func (h *GetTeacherGameDetailHandler) Handle(ctx context.Context, q GetTeacherGameDetailQuery) (*entities.TeacherGame, error) {
	if q.GameID == "" {
		return nil, errors.New("el identificador del juego es obligatorio")
	}

	game, err := h.gameRepo.FindByID(ctx, q.GameID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, apperrors.New("No se encontró la actividad.")
	}
	if q.TeacherID != "" && game.TeacherID != q.TeacherID {
		return nil, apperrors.New("No tienes permisos para acceder a esta actividad.")
	}

	return game, nil
}
