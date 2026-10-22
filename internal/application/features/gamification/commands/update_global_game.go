package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// UpdateGlobalGameCommand edits a platform-global activity. CourseID is not
// part of the contract: a global activity is never linked to a single course.
type UpdateGlobalGameCommand struct {
	GameID           string  `json:"-"`
	AuthorID         string  `json:"-"`
	BaseArchetype    string  `json:"base_archetype"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	Level            string  `json:"level"`
	TimeLimitSeconds *int    `json:"time_limit_seconds"`
	AllowedMistakes  *int    `json:"allowed_mistakes"`
	RewardXP         *int    `json:"reward_xp"`
	ItemCount        *int    `json:"item_count"`
	SourceType       string  `json:"source_type"`
	SourceID         *string `json:"source_id"`
	CustomWordsJSON  *string `json:"custom_words_json"`
}

type UpdateGlobalGameHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewUpdateGlobalGameHandler(gameRepo repositories.TeacherGameRepository) *UpdateGlobalGameHandler {
	return &UpdateGlobalGameHandler{gameRepo: gameRepo}
}

// Handle applies a partial update over a platform-global activity. Authorization
// is the admin/moderator role gate at the route, so ownership is never compared
// here; instead, this handler refuses any activity whose owner_type is not the
// platform, which stops an administrator from editing a teacher's own activity
// through the admin endpoints. owner_type is the only ownership authority;
// is_global is legacy data and is never used for authorization.
func (h *UpdateGlobalGameHandler) Handle(ctx context.Context, cmd UpdateGlobalGameCommand) (*entities.TeacherGame, error) {
	if cmd.AuthorID == "" {
		return nil, errors.New("el identificador del administrador es obligatorio")
	}
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

	archetype := strings.ToLower(strings.TrimSpace(cmd.BaseArchetype))
	if archetype != "" {
		game.BaseArchetype = archetype
	}

	title := strings.TrimSpace(cmd.Title)
	if title != "" {
		game.Title = title
	}

	if cmd.Description != nil {
		trimmedDesc := strings.TrimSpace(*cmd.Description)
		if trimmedDesc == "" {
			game.Description = nil
		} else {
			game.Description = &trimmedDesc
		}
	}

	level := strings.TrimSpace(cmd.Level)
	if level != "" {
		game.Level = level
	}

	if cmd.TimeLimitSeconds != nil && *cmd.TimeLimitSeconds > 0 {
		game.TimeLimitSeconds = *cmd.TimeLimitSeconds
	}
	if cmd.AllowedMistakes != nil {
		game.AllowedMistakes = *cmd.AllowedMistakes
	}
	if cmd.RewardXP != nil && *cmd.RewardXP > 0 {
		game.RewardXP = *cmd.RewardXP
	}
	if cmd.ItemCount != nil && *cmd.ItemCount > 0 {
		game.ItemCount = *cmd.ItemCount
	}

	sourceType := strings.TrimSpace(cmd.SourceType)
	if sourceType != "" {
		game.SourceType = sourceType
	}
	if cmd.SourceID != nil {
		game.SourceID = cmd.SourceID
	}
	if cmd.CustomWordsJSON != nil {
		trimmed := strings.TrimSpace(*cmd.CustomWordsJSON)
		switch {
		case trimmed == "":
			empty := "[]"
			game.CustomWordsJSON = &empty
		case !json.Valid([]byte(trimmed)):
			return nil, errors.New("el campo custom_words_json no contiene un JSON válido")
		default:
			game.CustomWordsJSON = &trimmed
		}
	}

	game.UpdatedAt = time.Now()

	if err := h.gameRepo.Update(ctx, game); err != nil {
		return nil, err
	}

	return game, nil
}
