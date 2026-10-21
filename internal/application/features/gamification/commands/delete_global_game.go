package commands

import (
	"context"
	"errors"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type DeleteGlobalGameCommand struct {
	GameID string
}

// DeleteGlobalGameResult informa si la actividad se archivó en lugar de borrarse
// y cuántas lecciones la usan.
type DeleteGlobalGameResult struct {
	Archived   bool
	UsageCount int
}

type DeleteGlobalGameHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewDeleteGlobalGameHandler(gameRepo repositories.TeacherGameRepository) *DeleteGlobalGameHandler {
	return &DeleteGlobalGameHandler{gameRepo: gameRepo}
}

// Handle deletes a platform-global activity. Authorization is the admin/moderator
// role gate at the route, so ownership is never compared here; instead, this
// handler refuses any activity whose owner_type is not the platform, which stops
// an administrator from deleting a teacher's own activity through the admin
// endpoints. When the activity is linked to lessons it is archived instead of
// removed, so existing links keep working.
func (h *DeleteGlobalGameHandler) Handle(ctx context.Context, cmd DeleteGlobalGameCommand) (*DeleteGlobalGameResult, error) {
	if cmd.GameID == "" {
		return nil, errors.New("el identificador del juego es obligatorio")
	}

	game, err := h.gameRepo.FindByID(ctx, cmd.GameID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, apperrors.New("No se encontró la actividad.")
	}
	if game.OwnerType != entities.OwnerTypePlatform {
		return nil, apperrors.New("Solo se pueden gestionar actividades de la plataforma.")
	}

	usage, err := h.gameRepo.CountLessonUsage(ctx, cmd.GameID)
	if err != nil {
		return nil, err
	}

	if usage > 0 {
		game.IsActive = false
		game.UpdatedAt = time.Now()
		if err := h.gameRepo.Update(ctx, game); err != nil {
			return nil, err
		}
		return &DeleteGlobalGameResult{Archived: true, UsageCount: usage}, nil
	}

	if err := h.gameRepo.Delete(ctx, cmd.GameID); err != nil {
		return nil, err
	}
	return &DeleteGlobalGameResult{Archived: false, UsageCount: 0}, nil
}
