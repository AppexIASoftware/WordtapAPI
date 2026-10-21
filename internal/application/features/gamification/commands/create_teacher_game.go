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

type CreateTeacherGameCommand struct {
	TeacherID        string  `json:"-"`
	CourseID         *string `json:"course_id"`
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

type CreateTeacherGameHandler struct {
	gameRepo repositories.TeacherGameRepository
}

func NewCreateTeacherGameHandler(gameRepo repositories.TeacherGameRepository) *CreateTeacherGameHandler {
	return &CreateTeacherGameHandler{gameRepo: gameRepo}
}

func (h *CreateTeacherGameHandler) Handle(ctx context.Context, cmd CreateTeacherGameCommand) (*entities.TeacherGame, error) {
	if cmd.TeacherID == "" {
		return nil, errors.New("el identificador del docente es obligatorio")
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

	var courseID *string
	if cmd.CourseID != nil && strings.TrimSpace(*cmd.CourseID) != "" {
		trimmed := strings.TrimSpace(*cmd.CourseID)
		courseID = &trimmed
	}

	var customWordsJSON *string
	if cmd.CustomWordsJSON != nil && strings.TrimSpace(*cmd.CustomWordsJSON) != "" {
		trimmed := strings.TrimSpace(*cmd.CustomWordsJSON)
		if json.Valid([]byte(trimmed)) {
			customWordsJSON = &trimmed
		} else {
			empty := "[]"
			customWordsJSON = &empty
		}
	}

	game := &entities.TeacherGame{
		TeacherID:        cmd.TeacherID,
		CourseID:         courseID,
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
		OwnerType:        entities.OwnerTypeTeacher,
		IsGlobal:         false,
		IsActive:         true,
	}

	if err := h.gameRepo.Create(ctx, game); err != nil {
		return nil, err
	}

	return game, nil
}
