package course

import (
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// PublicLessonDTO representa la vista pública de una lección sin tarjetas internas.
type PublicLessonDTO struct {
	ID               string                 `json:"id"`
	CourseID         string                 `json:"course_id"`
	Title            string                 `json:"title"`
	Slug             string                 `json:"slug"`
	Description      *string                `json:"description,omitempty"`
	AccessTier       entities.AccessTier    `json:"access_tier"`
	Status           entities.ContentStatus `json:"status"`
	SortOrder        int                    `json:"sort_order"`
	EstimatedMinutes *int                   `json:"estimated_minutes,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// PublicCourseDTO proyecta un curso para catálogo y detalle público anónimo sin datos de autor ni tarjetas privadas.
type PublicCourseDTO struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Slug          string                 `json:"slug"`
	Description   *string                `json:"description,omitempty"`
	AccessTier    entities.AccessTier    `json:"access_tier"`
	PriceCents    int                    `json:"price_cents"`
	Level         string                 `json:"level"`
	Status        entities.ContentStatus `json:"status"`
	SortOrder     int                    `json:"sort_order"`
	CoverImageURL *string                `json:"cover_image_url,omitempty"`
	SourceLang    string                 `json:"source_lang"`
	TargetLang    string                 `json:"target_lang"`
	LessonsCount  int                    `json:"lessons_count"`
	Lessons       []PublicLessonDTO      `json:"lessons,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ToPublicCourseDTO convierte una entidad Course en un DTO público sanitizado.
func ToPublicCourseDTO(c *entities.Course) PublicCourseDTO {
	publicLessons := make([]PublicLessonDTO, 0)
	for _, l := range c.Lessons {
		if l.Status == entities.ContentStatusPublished {
			publicLessons = append(publicLessons, PublicLessonDTO{
				ID:               l.ID,
				CourseID:         l.CourseID,
				Title:            l.Title,
				Slug:             l.Slug,
				Description:      l.Description,
				AccessTier:       l.AccessTier,
				Status:           l.Status,
				SortOrder:        l.SortOrder,
				EstimatedMinutes: l.EstimatedMinutes,
				CreatedAt:        l.CreatedAt,
				UpdatedAt:        l.UpdatedAt,
			})
		}
	}

	return PublicCourseDTO{
		ID:            c.ID,
		Title:         c.Title,
		Slug:          c.Slug,
		Description:   c.Description,
		AccessTier:    c.AccessTier,
		PriceCents:    c.PriceCents,
		Level:         c.Level,
		Status:        c.Status,
		SortOrder:     c.SortOrder,
		CoverImageURL: c.CoverImageURL,
		SourceLang:    c.SourceLang,
		TargetLang:    c.TargetLang,
		LessonsCount:  len(publicLessons),
		Lessons:       publicLessons,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}
