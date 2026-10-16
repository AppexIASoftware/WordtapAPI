package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListTeacherGamesQuery struct {
	TeacherID string
}

type ListTeacherGamesHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewListTeacherGamesHandler(gameRepo repositories.TeacherGameRepository) *ListTeacherGamesHandler {
	return &ListTeacherGamesHandler{gameRepo: gameRepo}
}

func (h *ListTeacherGamesHandler) Handle(ctx context.Context, q ListTeacherGamesQuery) ([]entities.TeacherGame, error) {
	if q.TeacherID == "" {
		return nil, errors.New("el identificador del docente es obligatorio")
	}
	return h.gameRepo.FindByTeacher(ctx, q.TeacherID)
}
