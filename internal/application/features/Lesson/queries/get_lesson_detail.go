package queries

import (
	"context"
	"fmt"

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

	dto := &LessonDetailDTO{
		ID:           lesson.ID,
		Title:        lesson.Title,
		AccessTier:   string(lesson.AccessTier),
		SectionTitle: "Verbos esenciales",
		Intro:        "Los verbos son el corazón del inglés. Estos verbos representan la base fundamental de las conversaciones cotidianas.",
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
			itemDTO.Note = *item.ContentText
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
