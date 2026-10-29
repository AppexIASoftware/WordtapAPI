package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// GetLessonDetailQuery representa el parámetro de consulta para obtener el detalle de una lección.
type GetLessonDetailQuery struct {
	Identifier string
}

// LessonItemDTO representa un elemento o tarjeta de estudio dentro de la lección para el cliente móvil.
type LessonItemDTO struct {
	ID                 string `json:"id"`
	BadgeType          string `json:"badgeType"`
	BadgeValue         string `json:"badgeValue"`
	Title              string `json:"title"`
	Translation        string `json:"translation"`
	ExampleOrPronounce string `json:"exampleOrPronounce"`
	Note               string `json:"note"`
	NoteType           string `json:"noteType"`
}

// LessonDetailDTO es la respuesta estructurada consumida por la aplicación móvil.
type LessonDetailDTO struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Subtitle     string          `json:"subtitle"`
	Intro        string          `json:"intro"`
	SectionTitle string          `json:"sectionTitle"`
	AccessTier   string          `json:"accessTier"`
	CategoryName string          `json:"categoryName"`
	Items        []LessonItemDTO `json:"items"`
}

// GetLessonDetailHandler procesa la consulta de detalle de lección (CQRS Query Handler).
type GetLessonDetailHandler struct {
	repo repositories.LessonRepository
}

// NewGetLessonDetailHandler inicializa el manejador de la consulta.
func NewGetLessonDetailHandler(repo repositories.LessonRepository) *GetLessonDetailHandler {
	return &GetLessonDetailHandler{repo: repo}
}

// Handle ejecuta la búsqueda y transforma la entidad a un DTO desacoplado para la app móvil.
func (h *GetLessonDetailHandler) Handle(ctx context.Context, query GetLessonDetailQuery) (*LessonDetailDTO, error) {
	lesson, err := h.repo.FindByIDOrSlug(ctx, query.Identifier)
	if err != nil {
		return nil, err
	}

	introText := ""
	subtitleText := "Domina los conceptos esenciales"
	if lesson.Description != nil && strings.TrimSpace(*lesson.Description) != "" {
		introText = strings.TrimSpace(*lesson.Description)
		subtitleText = strings.TrimSpace(*lesson.Description)
	}

	dto := &LessonDetailDTO{
		ID:           lesson.ID,
		Title:        lesson.Title,
		Subtitle:     subtitleText,
		AccessTier:   string(lesson.AccessTier),
		SectionTitle: "Tarjetas de estudio",
		Intro:        introText,
		Items:        make([]LessonItemDTO, 0, len(lesson.Items)),
	}

	if lesson.Description != nil {
		dto.Subtitle = *lesson.Description
	}

	if lesson.Category != nil {
		dto.CategoryName = lesson.Category.Name
	}

	for idx, item := range lesson.Items {
		itemDTO := LessonItemDTO{
			ID:         item.ID,
			BadgeType:  "number",
			BadgeValue: fmt.Sprintf("%d", idx+1),
			NoteType:   "bulb",
		}

		if item.ContentText != nil {
			var cardPayload struct {
				EN        string `json:"en"`
				ES        string `json:"es"`
				Title     string `json:"title"`
				IPA       string `json:"ipa"`
				Example   string `json:"example"`
				Note      string `json:"note"`
				NoteType  string `json:"noteType"`
				BadgeType string `json:"badgeType"`
			}
			if err := json.Unmarshal([]byte(*item.ContentText), &cardPayload); err == nil && (cardPayload.EN != "" || cardPayload.ES != "" || cardPayload.Title != "") {
				if cardPayload.EN != "" {
					itemDTO.Title = cardPayload.EN
				} else if cardPayload.Title != "" {
					itemDTO.Title = cardPayload.Title
				}
				if cardPayload.ES != "" {
					itemDTO.Translation = cardPayload.ES
				}
				if cardPayload.Example != "" {
					itemDTO.ExampleOrPronounce = cardPayload.Example
				} else if cardPayload.IPA != "" {
					itemDTO.ExampleOrPronounce = cardPayload.IPA
				}
				if cardPayload.Note != "" {
					itemDTO.Note = cardPayload.Note
				}
				if cardPayload.NoteType != "" {
					itemDTO.NoteType = cardPayload.NoteType
				}
				if cardPayload.BadgeType != "" {
					itemDTO.BadgeType = cardPayload.BadgeType
				}
			} else {
				itemDTO.Note = *item.ContentText
			}
		}

		if item.VocabularyItem != nil {
			itemDTO.Title = item.VocabularyItem.English
			itemDTO.Translation = item.VocabularyItem.Spanish
			if item.VocabularyItem.ExampleSentence != nil {
				itemDTO.ExampleOrPronounce = *item.VocabularyItem.ExampleSentence
			} else if item.VocabularyItem.Pronunciation != nil {
				itemDTO.ExampleOrPronounce = *item.VocabularyItem.Pronunciation
			}
			if item.VocabularyItem.Definition != nil && itemDTO.Note == "" {
				itemDTO.Note = *item.VocabularyItem.Definition
			}
		} else if item.Phrase != nil {
			itemDTO.Title = item.Phrase.English
			itemDTO.Translation = item.Phrase.Spanish
			if item.Phrase.Explanation != nil && itemDTO.Note == "" {
				itemDTO.Note = *item.Phrase.Explanation
			}
		}

		dto.Items = append(dto.Items, itemDTO)
	}

	return dto, nil
}
