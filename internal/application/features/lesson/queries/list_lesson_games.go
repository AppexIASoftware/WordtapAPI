package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	lessonAccess "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrLessonGamesNotFound     = fmt.Errorf("%w", apperrors.New("No se encontró la lección."))
	ErrLessonGamesAccessDenied = fmt.Errorf("%w", apperrors.New("No tienes permisos para ver los juegos de esta lección."))
)

// LessonGameWordDTO represents a playable word pair.
type LessonGameWordDTO struct {
	ID          string `json:"id,omitempty"`
	Term        string `json:"term"`
	Translation string `json:"translation"`
	English     string `json:"en,omitempty"`
	Spanish     string `json:"es,omitempty"`
}

// LessonGameDTO projects a lesson-game link.
type LessonGameDTO struct {
	ID               string              `json:"id"`
	GameID           string              `json:"game_id"`
	Title            string              `json:"title"`
	Description      *string             `json:"description,omitempty"`
	BaseArchetype    string              `json:"base_archetype"`
	Level            string              `json:"level"`
	RewardXP         int                 `json:"reward_xp"`
	TimeLimitSeconds int                 `json:"time_limit_seconds"`
	AllowedMistakes  int                 `json:"allowed_mistakes"`
	ItemCount        int                 `json:"item_count"`
	OwnerType        entities.OwnerType  `json:"owner_type"`
	IsGlobal         bool                `json:"is_global"`
	IsActive         bool                `json:"is_active"`
	IsRequired       bool                `json:"is_required"`
	SortOrder        int                 `json:"sort_order"`
	UsageCount       int                 `json:"usage_count"`
	LevelWarning     string              `json:"level_warning,omitempty"`
	Words            []LessonGameWordDTO `json:"words"`
}

// ToLessonGameDTO converts an entity link to DTO.
func ToLessonGameDTO(link entities.LessonGame) LessonGameDTO {
	dto := LessonGameDTO{
		ID:         link.ID,
		GameID:     link.TeacherGameID,
		IsRequired: link.IsRequired,
		SortOrder:  link.SortOrder,
		Words:      make([]LessonGameWordDTO, 0),
	}
	if link.TeacherGame == nil {
		return dto
	}

	game := link.TeacherGame
	dto.Title = game.Title
	dto.Description = game.Description
	dto.BaseArchetype = game.BaseArchetype
	dto.Level = game.Level
	dto.RewardXP = game.RewardXP
	dto.TimeLimitSeconds = game.TimeLimitSeconds
	dto.AllowedMistakes = game.AllowedMistakes
	dto.ItemCount = game.ItemCount
	dto.OwnerType = game.OwnerType
	dto.IsGlobal = game.IsGlobal
	dto.IsActive = game.IsActive
	dto.UsageCount = game.UsageCount
	dto.LevelWarning = link.LevelWarning
	return dto
}

// extractGameWords extracts words from lesson cards with custom words fallback.
func extractGameWords(lesson *entities.Lesson, game *entities.TeacherGame) []LessonGameWordDTO {
	words := make([]LessonGameWordDTO, 0)
	if lesson != nil && len(lesson.Items) > 0 {
		for _, item := range lesson.Items {
			var term, translation, id string
			if item.VocabularyItem != nil {
				term = strings.TrimSpace(item.VocabularyItem.English)
				translation = strings.TrimSpace(item.VocabularyItem.Spanish)
				id = item.VocabularyItem.ID
			} else if item.Phrase != nil {
				term = strings.TrimSpace(item.Phrase.English)
				translation = strings.TrimSpace(item.Phrase.Spanish)
				id = item.Phrase.ID
			} else if item.ContentBankItem != nil {
				term = strings.TrimSpace(item.ContentBankItem.TargetText)
				translation = strings.TrimSpace(item.ContentBankItem.PromptText)
				id = item.ContentBankItem.ID
			}
			if id == "" {
				id = item.ID
			}
			if term != "" && translation != "" {
				words = append(words, LessonGameWordDTO{
					ID:          id,
					Term:        term,
					Translation: translation,
					English:     term,
					Spanish:     translation,
				})
			}
		}
	}

	if len(words) == 0 && game != nil && game.CustomWordsJSON != nil && strings.TrimSpace(*game.CustomWordsJSON) != "" {
		var raw []struct {
			ID          string `json:"id"`
			Term        string `json:"term"`
			Translation string `json:"translation"`
			English     string `json:"en"`
			Spanish     string `json:"es"`
		}
		if err := json.Unmarshal([]byte(*game.CustomWordsJSON), &raw); err == nil {
			for i, r := range raw {
				term := strings.TrimSpace(r.Term)
				if term == "" {
					term = strings.TrimSpace(r.English)
				}
				trans := strings.TrimSpace(r.Translation)
				if trans == "" {
					trans = strings.TrimSpace(r.Spanish)
				}
				id := strings.TrimSpace(r.ID)
				if id == "" {
					id = fmt.Sprintf("custom-%d", i+1)
				}
				if term != "" && trans != "" {
					words = append(words, LessonGameWordDTO{
						ID:          id,
						Term:        term,
						Translation: trans,
						English:     term,
						Spanish:     trans,
					})
				}
			}
		}
	}

	if game != nil && game.ItemCount > 0 && len(words) > game.ItemCount {
		words = words[:game.ItemCount]
	}

	return words
}

type ListLessonGamesQuery struct {
	LessonID      string
	RequesterID   string
	RequesterRole entities.UserRole
}

type ListLessonGamesHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
	linkRepo   repositories.LessonGameRepository
}

func NewListLessonGamesHandler(
	lessonRepo repositories.LessonRepository,
	courseRepo repositories.CourseRepository,
	linkRepo repositories.LessonGameRepository,
) *ListLessonGamesHandler {
	return &ListLessonGamesHandler{
		lessonRepo: lessonRepo,
		courseRepo: courseRepo,
		linkRepo:   linkRepo,
	}
}

func (h *ListLessonGamesHandler) Handle(ctx context.Context, query ListLessonGamesQuery) ([]LessonGameDTO, error) {
	if strings.TrimSpace(query.LessonID) == "" {
		return nil, ErrLessonGamesNotFound
	}

	lesson, err := h.lessonRepo.FindByID(ctx, query.LessonID)
	if err != nil || lesson == nil {
		return nil, ErrLessonGamesNotFound
	}

	course, err := h.courseRepo.FindByID(ctx, lesson.CourseID)
	if err != nil || course == nil {
		return nil, ErrLessonGamesNotFound
	}

	canManage := lessonAccess.CanManageCourse(course, query.RequesterID, query.RequesterRole)
	if !canManage {
		if course.Status != entities.ContentStatusPublished || lesson.Status != entities.ContentStatusPublished {
			return nil, ErrLessonGamesAccessDenied
		}
	}

	links, err := h.linkRepo.FindByLesson(ctx, query.LessonID)
	if err != nil {
		return nil, err
	}

	visible := make([]entities.LessonGame, 0, len(links))
	for _, link := range links {
		if link.TeacherGame == nil {
			continue
		}
		if !canManage && !link.TeacherGame.IsActive {
			continue
		}
		visible = append(visible, link)
	}

	ids := make([]string, 0, len(visible))
	for _, link := range visible {
		ids = append(ids, link.TeacherGameID)
	}

	usage, err := h.linkRepo.CountByGameIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	dtos := make([]LessonGameDTO, 0, len(visible))
	for _, link := range visible {
		dto := ToLessonGameDTO(link)
		dto.UsageCount = usage[link.TeacherGameID]
		dto.LevelWarning = entities.LevelWarningMessage(
			entities.CompareLevels(course.Level, link.TeacherGame.Level),
		)
		dto.Words = extractGameWords(lesson, link.TeacherGame)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}
