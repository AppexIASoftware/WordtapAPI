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

// PublicAuthorDTO representa los datos públicos del autor/docente de un curso.
type PublicAuthorDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// PublicCourseDTO proyecta un curso para catálogo y detalle público con autor sanitizado.
type PublicCourseDTO struct {
	ID                    string                 `json:"id"`
	Title                 string                 `json:"title"`
	Slug                  string                 `json:"slug"`
	Description           *string                `json:"description,omitempty"`
	AccessTier            entities.AccessTier    `json:"access_tier"`
	PriceCents            int                    `json:"price_cents"`
	Level                 string                 `json:"level"`
	Status                entities.ContentStatus `json:"status"`
	OwnerType             entities.OwnerType     `json:"owner_type"`
	SortOrder             int                    `json:"sort_order"`
	CoverImageURL         *string                `json:"cover_image_url,omitempty"`
	CategoryID            *string                `json:"category_id,omitempty"`
	CategoryName          *string                `json:"category_name,omitempty"`
	SourceLang            string                 `json:"source_lang"`
	TargetLang            string                 `json:"target_lang"`
	Author                *PublicAuthorDTO       `json:"author,omitempty"`
	LessonsCount          int                    `json:"lessons_count"`
	TotalEstimatedMinutes int                    `json:"total_estimated_minutes"`
	Lessons               []PublicLessonDTO      `json:"lessons,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// ToPublicCourseDTO convierte una entidad Course en un DTO público sanitizado.
func ToPublicCourseDTO(c *entities.Course) PublicCourseDTO {
	publicLessons := make([]PublicLessonDTO, 0)
	totalMinutes := 0
	for _, l := range c.Lessons {
		if l.Status == entities.ContentStatusPublished {
			mins := 15
			if l.EstimatedMinutes != nil && *l.EstimatedMinutes > 0 {
				mins = *l.EstimatedMinutes
			}
			totalMinutes += mins
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

	var authorDTO *PublicAuthorDTO
	if c.Author != nil {
		authorDTO = &PublicAuthorDTO{
			ID:        c.Author.ID,
			Name:      c.Author.Name,
			Role:      string(c.Author.Role),
			AvatarURL: c.Author.AvatarURL,
		}
	}

	var catName *string
	if c.Category != nil {
		catName = &c.Category.Name
	}

	return PublicCourseDTO{
		ID:                    c.ID,
		Title:                 c.Title,
		Slug:                  c.Slug,
		Description:           c.Description,
		AccessTier:            c.AccessTier,
		PriceCents:            c.PriceCents,
		Level:                 c.Level,
		CategoryID:            c.CategoryID,
		CategoryName:          catName,
		Status:                c.Status,
		OwnerType:             c.OwnerType,
		SortOrder:             c.SortOrder,
		CoverImageURL:         c.CoverImageURL,
		SourceLang:            c.SourceLang,
		TargetLang:            c.TargetLang,
		Author:                authorDTO,
		LessonsCount:          len(publicLessons),
		TotalEstimatedMinutes: totalMinutes,
		Lessons:               publicLessons,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
	}
}
