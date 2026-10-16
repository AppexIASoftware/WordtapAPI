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

type UpdateTeacherGameCommand struct {
	GameID           string  `json:"-"`
	TeacherID        string  `json:"-"`
	CourseID         *string `json:"course_id"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	Level            string  `json:"level"`
	TimeLimitSeconds int     `json:"time_limit_seconds"`
	AllowedMistakes  int     `json:"allowed_mistakes"`
	RewardXP         int     `json:"reward_xp"`
	ItemCount        int     `json:"item_count"`
	SourceType       string  `json:"source_type"`
	SourceID         *string `json:"source_id"`
	CustomWordsJSON  *string `json:"custom_words_json"`
}

type UpdateTeacherGameHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewUpdateTeacherGameHandler(gameRepo repositories.TeacherGameRepository) *UpdateTeacherGameHandler {
	return &UpdateTeacherGameHandler{gameRepo: gameRepo}
}

func (h *UpdateTeacherGameHandler) Handle(ctx context.Context, cmd UpdateTeacherGameCommand) (*entities.TeacherGame, error) {
	if cmd.TeacherID == "" {
		return nil, errors.New("el identificador del docente es obligatorio")
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
	if game.TeacherID != cmd.TeacherID {
		return nil, apperrors.New("No tienes permisos para editar esta actividad.")
	}

	title := strings.TrimSpace(cmd.Title)
	if title != "" {
		game.Title = title
	}

	if cmd.CourseID != nil {
		if strings.TrimSpace(*cmd.CourseID) == "" {
			game.CourseID = nil
		} else {
			trimmed := strings.TrimSpace(*cmd.CourseID)
			game.CourseID = &trimmed
		}
	}
	if cmd.Description != nil {
		game.Description = cmd.Description
	}

	level := strings.TrimSpace(cmd.Level)
	if level != "" {
		game.Level = level
	}

	if cmd.TimeLimitSeconds > 0 {
		game.TimeLimitSeconds = cmd.TimeLimitSeconds
	}
	if cmd.AllowedMistakes >= 0 {
		game.AllowedMistakes = cmd.AllowedMistakes
	}
	if cmd.RewardXP > 0 {
		game.RewardXP = cmd.RewardXP
	}
	if cmd.ItemCount > 0 {
		game.ItemCount = cmd.ItemCount
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
		if trimmed == "" || !json.Valid([]byte(trimmed)) {
			empty := "[]"
			game.CustomWordsJSON = &empty
		} else {
			game.CustomWordsJSON = &trimmed
		}
	}

	game.UpdatedAt = time.Now()

	if err := h.gameRepo.Update(ctx, game); err != nil {
		return nil, err
	}

	return game, nil
}
