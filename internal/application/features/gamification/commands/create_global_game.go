package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// CreateGlobalGameCommand creates a platform-global activity. Global activities
// belong to the platform itself, so they are never attached to a course.
type CreateGlobalGameCommand struct {
	AuthorID         string  `json:"-"`
	CategoryID       *string `json:"category_id"`
	BaseArchetype    string  `json:"base_archetype"`
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

type CreateGlobalGameHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewCreateGlobalGameHandler(gameRepo repositories.TeacherGameRepository) *CreateGlobalGameHandler {
	return &CreateGlobalGameHandler{gameRepo: gameRepo}
}

// Handle persists a platform-global activity. TeacherID keeps the identifier of
// the administrator who created it (the column is required) and is used for the
// author display only: a global activity belongs to the platform, not to that
// individual administrator.
func (h *CreateGlobalGameHandler) Handle(ctx context.Context, cmd CreateGlobalGameCommand) (*entities.TeacherGame, error) {
	if cmd.AuthorID == "" {
		return nil, errors.New("el identificador del administrador es obligatorio")
	}

	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, apperrors.New("El título de la actividad es obligatorio.")
	}

	archetype := strings.ToLower(strings.TrimSpace(cmd.BaseArchetype))
	if archetype == "" {
		archetype = "match"
	}

	level := strings.TrimSpace(cmd.Level)
	if level == "" {
		level = "B1 Intermediate"
	}

	timeLimit := cmd.TimeLimitSeconds
	if timeLimit <= 0 {
		timeLimit = 45
	}

	allowedMistakes := cmd.AllowedMistakes
	if allowedMistakes < 0 {
		allowedMistakes = 3
	}

	rewardXP := cmd.RewardXP
	if rewardXP <= 0 {
		rewardXP = 20
	}

	itemCount := cmd.ItemCount
	if itemCount <= 0 {
		itemCount = 6
	}

	sourceType := strings.TrimSpace(cmd.SourceType)
	if sourceType == "" {
		sourceType = "course_lesson"
	}

	var categoryID *string
	if cmd.CategoryID != nil && strings.TrimSpace(*cmd.CategoryID) != "" {
		trimmed := strings.TrimSpace(*cmd.CategoryID)
		categoryID = &trimmed
	}

	var customWordsJSON *string
	if cmd.CustomWordsJSON != nil && strings.TrimSpace(*cmd.CustomWordsJSON) != "" {
		trimmed := strings.TrimSpace(*cmd.CustomWordsJSON)
		if !json.Valid([]byte(trimmed)) {
			return nil, errors.New("el campo custom_words_json no contiene un JSON válido")
		}
		customWordsJSON = &trimmed
	}

	game := &entities.TeacherGame{
		TeacherID:        cmd.AuthorID,
		CourseID:         nil,
		CategoryID:       categoryID,
		BaseArchetype:    archetype,
		Title:            title,
		Description:      cmd.Description,
		Level:            level,
		TimeLimitSeconds: timeLimit,
		AllowedMistakes:  allowedMistakes,
		RewardXP:         rewardXP,
		ItemCount:        itemCount,
		SourceType:       sourceType,
		SourceID:         cmd.SourceID,
		CustomWordsJSON:  customWordsJSON,
		OwnerType:        entities.OwnerTypePlatform,
		IsGlobal:         true,
		IsActive:         true,
	}

	if err := h.gameRepo.Create(ctx, game); err != nil {
		return nil, err
	}

	return game, nil
}
