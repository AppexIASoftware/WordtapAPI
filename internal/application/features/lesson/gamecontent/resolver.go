// Package gamecontent resolves the word pairs exposed for a lesson-linked game.
package gamecontent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// Word is a playable pair, including the legacy language aliases.
type Word struct {
	ID          string `json:"id,omitempty"`
	Term        string `json:"term"`
	Translation string `json:"translation"`
	English     string `json:"en,omitempty"`
	Spanish     string `json:"es,omitempty"`
}

// Resolve uses a global game's own content. Other games use lesson cards first,
// with the legacy custom-word fallback when there are no valid cards.
func Resolve(lesson *entities.Lesson, game *entities.TeacherGame) []Word {
	words := make([]Word, 0)
	if lesson != nil && (game == nil || !game.IsGlobal) {
		items := append([]entities.LessonItem(nil), lesson.Items...)
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].SortOrder != items[j].SortOrder {
				return items[i].SortOrder < items[j].SortOrder
			}
			return items[i].ID < items[j].ID
		})
		for _, item := range items {
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
			} else if item.ContentText != nil && strings.TrimSpace(*item.ContentText) != "" {
				var card struct {
					EN          string `json:"en"`
					ES          string `json:"es"`
					Title       string `json:"title"`
					Term        string `json:"term"`
					Translation string `json:"translation"`
				}
				if json.Unmarshal([]byte(*item.ContentText), &card) == nil {
					if card.EN != "" {
						term = strings.TrimSpace(card.EN)
					} else if card.Term != "" {
						term = strings.TrimSpace(card.Term)
					} else if card.Title != "" {
						term = strings.TrimSpace(card.Title)
					}
					if card.ES != "" {
						translation = strings.TrimSpace(card.ES)
					} else if card.Translation != "" {
						translation = strings.TrimSpace(card.Translation)
					}
				}
			}
			if id == "" {
				id = item.ID
			}
			if term != "" && translation != "" {
				words = append(words, Word{ID: id, Term: term, Translation: translation, English: term, Spanish: translation})
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
		if json.Unmarshal([]byte(*game.CustomWordsJSON), &raw) == nil {
			for i, r := range raw {
				term := strings.TrimSpace(r.Term)
				if term == "" {
					term = strings.TrimSpace(r.English)
				}
				translation := strings.TrimSpace(r.Translation)
				if translation == "" {
					translation = strings.TrimSpace(r.Spanish)
				}
				id := strings.TrimSpace(r.ID)
				if id == "" {
					id = fmt.Sprintf("custom-%d", i+1)
				}
				if term != "" && translation != "" {
					words = append(words, Word{ID: id, Term: term, Translation: translation, English: term, Spanish: translation})
				}
			}
		}
	}
	if game != nil && game.ItemCount > 0 && len(words) > game.ItemCount {
		words = words[:game.ItemCount]
	}
	return words
}
