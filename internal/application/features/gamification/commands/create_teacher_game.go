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
	TeacherID             string              `json:"-"`
	OwnerType             entities.OwnerType  `json:"-"`
	CourseID              *string             `json:"course_id"`
	CategoryID            *string             `json:"category_id"`
	BaseArchetype         string              `json:"base_archetype"`
	Title                 string              `json:"title"`
	Description           *string             `json:"description"`
	Level                 string              `json:"level"`
	AccessTier            entities.AccessTier `json:"access_tier"`
	TimeLimitSeconds      int                 `json:"time_limit_seconds"`
	AllowedMistakes       int                 `json:"allowed_mistakes"`
	RewardXP              int                 `json:"reward_xp"`
	ItemCount             int                 `json:"item_count"`
	SourceType            string              `json:"source_type"`
	SourceID              *string             `json:"source_id"`
	CustomWordsJSON       *string             `json:"custom_words_json"`
	IsPublishedStandalone bool                `json:"is_published_standalone"`
	AudienceScope         string              `json:"audience_scope"`
	AssignedCourseIDs     []string            `json:"assigned_course_ids"`
}

type CreateTeacherGameHandler struct {
	gameRepo   repositories.TeacherGameRepository
	courseRepo audienceCourseRepository
}

func NewCreateTeacherGameHandler(gameRepo repositories.TeacherGameRepository, courseRepos ...audienceCourseRepository) *CreateTeacherGameHandler {
	h := &CreateTeacherGameHandler{gameRepo: gameRepo}
	if len(courseRepos) > 0 {
		h.courseRepo = courseRepos[0]
	}
	return h
}

func (h *CreateTeacherGameHandler) Handle(ctx context.Context, cmd CreateTeacherGameCommand) (*entities.TeacherGame, error) {
	if cmd.TeacherID == "" {
		return nil, errors.New("el identificador del docente es obligatorio")
	}
	if err := validateAudienceCourses(ctx, h.courseRepo, cmd.TeacherID, cmd.AssignedCourseIDs); err != nil {
		return nil, err
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

	var categoryID *string
	if cmd.CategoryID != nil && strings.TrimSpace(*cmd.CategoryID) != "" {
		trimmed := strings.TrimSpace(*cmd.CategoryID)
		categoryID = &trimmed
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

	ownerType := cmd.OwnerType
	if ownerType != entities.OwnerTypePlatform {
		ownerType = entities.OwnerTypeTeacher
	}
	accessTier := cmd.AccessTier
	if accessTier == "" {
		accessTier = entities.AccessTierFree
	}
	audienceScope := strings.TrimSpace(cmd.AudienceScope)
	if audienceScope != "selected_courses" {
		audienceScope = "all_students"
	}

	var assignedCourseIDs *string
	if len(cmd.AssignedCourseIDs) > 0 {
		raw, _ := json.Marshal(cmd.AssignedCourseIDs)
		str := string(raw)
		assignedCourseIDs = &str
	}

	game := &entities.TeacherGame{
		TeacherID:             cmd.TeacherID,
		CourseID:              courseID,
		CategoryID:            categoryID,
		BaseArchetype:         archetype,
		Title:                 title,
		Description:           cmd.Description,
		Level:                 level,
		AccessTier:            accessTier,
		TimeLimitSeconds:      timeLimit,
		AllowedMistakes:       allowedMistakes,
		RewardXP:              rewardXP,
		ItemCount:             itemCount,
		SourceType:            sourceType,
		SourceID:              cmd.SourceID,
		CustomWordsJSON:       customWordsJSON,
		OwnerType:             ownerType,
		IsGlobal:              false,
		IsPublishedStandalone: cmd.IsPublishedStandalone,
		AudienceScope:         audienceScope,
		AssignedCourseIDs:     assignedCourseIDs,
		IsActive:              true,
	}

	if err := h.gameRepo.Create(ctx, game); err != nil {
		return nil, err
	}

	return game, nil
}
