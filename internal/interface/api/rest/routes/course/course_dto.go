package course

import (
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// PublicLessonDTO represents the public view of a lesson without internal cards.
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
	Items            []entities.LessonItem  `json:"items,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// PublicAuthorDTO represents the public details of a course's author or teacher.
type PublicAuthorDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	Specialty *string `json:"specialty,omitempty"`
}

// PublicCourseDTO projects a course for the public catalog and detail view with a sanitized author.
type PublicCourseDTO struct {
	ID                    string                 `json:"id"`
	Title                 string                 `json:"title"`
	Slug                  string                 `json:"slug"`
	Description           *string                `json:"description,omitempty"`
	AccessTier            entities.AccessTier    `json:"access_tier"`
	PriceCents            int                    `json:"price_cents"`
	Currency              string                 `json:"currency"`
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

// ToPublicCourseDTO converts a Course entity into a sanitized public DTO.
func ToPublicCourseDTO(c *entities.Course) PublicCourseDTO {
	return toPublicCourseDTO(c, c.OwnerType == entities.OwnerTypePlatform)
}

func toPublicCourseDTO(c *entities.Course, includeLessons bool) PublicCourseDTO {
	return toCourseDetailDTO(c, includeLessons, false)
}

func toAuditCourseDTO(c *entities.Course) PublicCourseDTO {
	return toCourseDetailDTO(c, true, true)
}

func toCourseDetailDTO(c *entities.Course, includeLessons bool, includeDraftsAndItems bool) PublicCourseDTO {
	publicLessons := make([]PublicLessonDTO, 0)
	totalMinutes := 0
	lessonCount := 0
	for _, l := range c.Lessons {
		if includeDraftsAndItems || l.Status == entities.ContentStatusPublished {
			lessonCount++
			mins := 15
			if l.EstimatedMinutes != nil && *l.EstimatedMinutes > 0 {
				mins = *l.EstimatedMinutes
			}
			totalMinutes += mins
			if includeLessons {
				lessonDTO := PublicLessonDTO{
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
				}
				if includeDraftsAndItems {
					lessonDTO.Items = l.Items
				}
				publicLessons = append(publicLessons, lessonDTO)
			}
		}
	}

	var authorDTO *PublicAuthorDTO
	if c.Author != nil {
		authorDTO = &PublicAuthorDTO{
			ID:        c.Author.ID,
			Name:      c.Author.Name,
			Role:      string(c.Author.Role),
			AvatarURL: c.Author.AvatarURL,
			Bio:       c.Author.Bio,
			Specialty: c.Author.Specialty,
		}
	}

	var catName *string
	if c.Category != nil {
		catName = &c.Category.Name
	}

	currency := c.Currency
	if currency == "" {
		currency = entities.CurrencyPEN
	}

	return PublicCourseDTO{
		ID:                    c.ID,
		Title:                 c.Title,
		Slug:                  c.Slug,
		Description:           c.Description,
		AccessTier:            c.AccessTier,
		PriceCents:            c.PriceCents,
		Currency:              currency,
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
		LessonsCount:          lessonCount,
		TotalEstimatedMinutes: totalMinutes,
		Lessons:               publicLessons,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
	}
}
