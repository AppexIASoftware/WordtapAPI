package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ListGlobalGamesQuery asks for the platform-owned activities available to any
// course, regardless of the teacher who will link them.
type ListGlobalGamesQuery struct{}

type ListGlobalGamesHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewListGlobalGamesHandler(gameRepo repositories.TeacherGameRepository) *ListGlobalGamesHandler {
	return &ListGlobalGamesHandler{gameRepo: gameRepo}
}

func (h *ListGlobalGamesHandler) Handle(ctx context.Context, _ ListGlobalGamesQuery) ([]entities.TeacherGame, error) {
	return h.gameRepo.FindGlobal(ctx)
}
