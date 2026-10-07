package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type UpdateCourseCommand struct {
	CourseID      string
	Title         *string
	Description   *string
	Level         *string
	CoverImageURL *string
	AccessTier    *entities.AccessTier
	PriceCents    *int
	ChangeSummary *string
	RequesterID   string
	RequesterRole entities.UserRole
}

type UpdateCourseHandler struct {
	courseRepo repositories.CourseRepository
	reviewRepo repositories.CourseReviewRepository
}

func NewUpdateCourseHandler(courseRepo repositories.CourseRepository, reviewRepo ...repositories.CourseReviewRepository) *UpdateCourseHandler {
	var rRepo repositories.CourseReviewRepository
	if len(reviewRepo) > 0 {
		rRepo = reviewRepo[0]
	}
	return &UpdateCourseHandler{courseRepo: courseRepo, reviewRepo: rRepo}
}

func (h *UpdateCourseHandler) Handle(ctx context.Context, cmd UpdateCourseCommand) (*entities.Course, error) {
	if cmd.CourseID == "" {
		return nil, errors.New("course id is required")
	}

	course, err := h.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	// Ownership check: only author, admin or moderator can edit
	if cmd.RequesterRole != entities.RoleAdmin && cmd.RequesterRole != entities.RoleModerator {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return nil, errors.New("forbidden: cannot edit a course created by another author")
		}
	}

	var changed []string

	// Apply field updates and track changes
	if cmd.Title != nil {
		cleanTitle := strings.TrimSpace(*cmd.Title)
		if cleanTitle != "" && cleanTitle != course.Title {
			changed = append(changed, fmt.Sprintf("Título: '%s' -> '%s'", course.Title, cleanTitle))
			course.Title = cleanTitle
		}
	}

	if cmd.Description != nil {
		changed = append(changed, "Descripción actualizada")
		course.Description = cmd.Description
	}

	if cmd.Level != nil {
		cleanLevel := strings.TrimSpace(*cmd.Level)
		if cleanLevel != "" && cleanLevel != course.Level {
			changed = append(changed, fmt.Sprintf("Nivel: '%s' -> '%s'", course.Level, cleanLevel))
			course.Level = cleanLevel
		}
	}

	if cmd.CoverImageURL != nil {
		changed = append(changed, "Portada actualizada")
		course.CoverImageURL = cmd.CoverImageURL
	}

	if cmd.AccessTier != nil && *cmd.AccessTier != "" && *cmd.AccessTier != course.AccessTier {
		changed = append(changed, fmt.Sprintf("Tier: '%s' -> '%s'", course.AccessTier, *cmd.AccessTier))
		course.AccessTier = *cmd.AccessTier
	}

	if cmd.PriceCents != nil && *cmd.PriceCents != course.PriceCents {
		changed = append(changed, fmt.Sprintf("Precio: %d -> %d", course.PriceCents, *cmd.PriceCents))
		course.PriceCents = *cmd.PriceCents
	}

	now := time.Now()
	wasPublished := course.Status == entities.ContentStatusPublished
	isTeacher := cmd.RequesterRole == entities.RoleInstructor

	// If published course is updated by teacher, transition back to in_review and generate approval request
	if wasPublished && isTeacher && len(changed) > 0 {
		course.Status = entities.ContentStatusInReview
	}

	course.UpdatedAt = now

	if err := h.courseRepo.Update(ctx, course); err != nil {
		return nil, fmt.Errorf("failed to update course: %w", err)
	}

	if wasPublished && isTeacher && h.reviewRepo != nil && len(changed) > 0 {
		summaryText := strings.Join(changed, "; ")
		if cmd.ChangeSummary != nil && strings.TrimSpace(*cmd.ChangeSummary) != "" {
			summaryText = fmt.Sprintf("%s (%s)", *cmd.ChangeSummary, summaryText)
		}
		diffJSON, _ := json.Marshal(map[string]any{
			"fields_changed": changed,
			"updated_at":     now,
		})
		reviewReq := &entities.CourseReviewRequest{
			CourseID:      course.ID,
			InstructorID:  cmd.RequesterID,
			Status:        entities.CourseReviewStatusPending,
			ChangeSummary: &summaryText,
			DiffSummary:   string(diffJSON),
			SubmittedAt:   now,
		}
		_ = h.reviewRepo.CreateSubmission(ctx, reviewReq)
	}

	return course, nil
}
