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
	GameID                string               `json:"-"`
	TeacherID             string               `json:"-"`
	CourseID              *string              `json:"course_id"`
	CategoryID            *string              `json:"category_id"`
	BaseArchetype         string               `json:"base_archetype"`
	Title                 string               `json:"title"`
	Description           *string              `json:"description"`
	Level                 string               `json:"level"`
	AccessTier            *entities.AccessTier `json:"access_tier"`
	TimeLimitSeconds      int                  `json:"time_limit_seconds"`
	AllowedMistakes       int                  `json:"allowed_mistakes"`
	RewardXP              int                  `json:"reward_xp"`
	ItemCount             int                  `json:"item_count"`
	SourceType            string               `json:"source_type"`
	SourceID              *string              `json:"source_id"`
	CustomWordsJSON       *string              `json:"custom_words_json"`
	IsPublishedStandalone *bool                `json:"is_published_standalone"`
	AudienceScope         *string              `json:"audience_scope"`
	AssignedCourseIDs     *[]string            `json:"assigned_course_ids"`
}

type UpdateTeacherGameHandler struct {
	gameRepo   repositories.TeacherGameRepository
	courseRepo audienceCourseRepository
}

func NewUpdateTeacherGameHandler(gameRepo repositories.TeacherGameRepository, courseRepos ...audienceCourseRepository) *UpdateTeacherGameHandler {
	h := &UpdateTeacherGameHandler{gameRepo: gameRepo}
	if len(courseRepos) > 0 {
		h.courseRepo = courseRepos[0]
	}
	return h
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
	if game.OwnerType == entities.OwnerTypePlatform {
		return nil, apperrors.New("Solo puedes editar tus propias actividades.")
	}
	if game.TeacherID != cmd.TeacherID {
		return nil, apperrors.New("No tienes permisos para editar esta actividad.")
	}
	if cmd.AssignedCourseIDs != nil {
		if err := validateAudienceCourses(ctx, h.courseRepo, cmd.TeacherID, *cmd.AssignedCourseIDs); err != nil {
			return nil, err
		}
	} else if game.AssignedCourseIDs != nil {
		var ids []string
		if err := json.Unmarshal([]byte(*game.AssignedCourseIDs), &ids); err != nil {
			return nil, err
		}
		if err := validateAudienceCourses(ctx, h.courseRepo, cmd.TeacherID, ids); err != nil {
			return nil, err
		}
	}
	if game.SourceType == "course_lesson" && game.SourceID != nil && strings.TrimSpace(*game.SourceID) != "" {
		if (strings.TrimSpace(cmd.SourceType) != "" && strings.TrimSpace(cmd.SourceType) != game.SourceType) ||
			(cmd.SourceID != nil && strings.TrimSpace(*cmd.SourceID) != *game.SourceID) {
			return nil, apperrors.New("No se puede cambiar la lección de esta actividad.")
		}
	}

	archetype := strings.ToLower(strings.TrimSpace(cmd.BaseArchetype))
	if archetype != "" {
		game.BaseArchetype = archetype
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
	if cmd.CategoryID != nil {
		if strings.TrimSpace(*cmd.CategoryID) == "" {
			game.CategoryID = nil
		} else {
			trimmed := strings.TrimSpace(*cmd.CategoryID)
			game.CategoryID = &trimmed
		}
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

	if cmd.AccessTier != nil && *cmd.AccessTier != "" {
		game.AccessTier = *cmd.AccessTier
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

	if cmd.IsPublishedStandalone != nil {
		game.IsPublishedStandalone = *cmd.IsPublishedStandalone
	}
	if cmd.AudienceScope != nil {
		scope := strings.TrimSpace(*cmd.AudienceScope)
		if scope != "selected_courses" {
			scope = "all_students"
		}
		game.AudienceScope = scope
	}
	if cmd.AssignedCourseIDs != nil {
		if len(*cmd.AssignedCourseIDs) == 0 {
			empty := "[]"
			game.AssignedCourseIDs = &empty
		} else {
			raw, _ := json.Marshal(*cmd.AssignedCourseIDs)
			str := string(raw)
			game.AssignedCourseIDs = &str
		}
	}

	game.UpdatedAt = time.Now()

	if err := h.gameRepo.Update(ctx, game); err != nil {
		return nil, err
	}

	return game, nil
}
