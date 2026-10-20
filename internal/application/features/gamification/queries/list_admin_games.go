package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListAdminGamesQuery struct{}

type ListAdminGamesHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewListAdminGamesHandler(gameRepo repositories.TeacherGameRepository) *ListAdminGamesHandler {
	return &ListAdminGamesHandler{gameRepo: gameRepo}
}

func (h *ListAdminGamesHandler) Handle(ctx context.Context, _ ListAdminGamesQuery) ([]entities.TeacherGame, error) {
	return h.gameRepo.FindAll(ctx)
}
