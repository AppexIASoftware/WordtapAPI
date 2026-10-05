package commands

import (
	"context"
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
	RequesterID   string
	RequesterRole entities.UserRole
}

type UpdateCourseHandler struct {
	courseRepo repositories.CourseRepository
}

func NewUpdateCourseHandler(courseRepo repositories.CourseRepository) *UpdateCourseHandler {
	return &UpdateCourseHandler{courseRepo: courseRepo}
}

func (h *UpdateCourseHandler) Handle(ctx context.Context, cmd UpdateCourseCommand) (*entities.Course, error) {
	if cmd.CourseID == "" {
		return nil, errors.New("course id is required")
	}

	course, err := h.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	// Ownership check: only author or admin can edit
	if cmd.RequesterRole != entities.RoleAdmin {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return nil, errors.New("forbidden: cannot edit a course created by another author")
		}
	}

	// Apply field updates
	if cmd.Title != nil {
		cleanTitle := strings.TrimSpace(*cmd.Title)
		if cleanTitle != "" {
			course.Title = cleanTitle
		}
	}

	if cmd.Description != nil {
		course.Description = cmd.Description
	}

	if cmd.Level != nil {
		cleanLevel := strings.TrimSpace(*cmd.Level)
		if cleanLevel != "" {
			course.Level = cleanLevel
		}
	}

	if cmd.CoverImageURL != nil {
		course.CoverImageURL = cmd.CoverImageURL
	}

	if cmd.AccessTier != nil && *cmd.AccessTier != "" {
		course.AccessTier = *cmd.AccessTier
	}

	if cmd.PriceCents != nil {
		course.PriceCents = *cmd.PriceCents
	}

	course.UpdatedAt = time.Now()

	if err := h.courseRepo.Update(ctx, course); err != nil {
		return nil, fmt.Errorf("failed to update course: %w", err)
	}

	return course, nil
}
