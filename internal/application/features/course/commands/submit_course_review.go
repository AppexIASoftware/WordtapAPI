package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type SubmitCourseReviewCommand struct {
	CourseID      string
	RequesterID   string
	RequesterRole entities.UserRole
}

type SubmitCourseReviewHandler struct {
	courseRepo repositories.CourseRepository
}

func NewSubmitCourseReviewHandler(courseRepo repositories.CourseRepository) *SubmitCourseReviewHandler {
	return &SubmitCourseReviewHandler{courseRepo: courseRepo}
}

func (h *SubmitCourseReviewHandler) Handle(ctx context.Context, cmd SubmitCourseReviewCommand) (*entities.Course, error) {
	if cmd.CourseID == "" {
		return nil, errors.New("course id is required")
	}

	course, err := h.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	// Ownership validation: only author or admin can submit for review
	if cmd.RequesterRole != entities.RoleAdmin {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return nil, errors.New("forbidden: cannot submit a course created by another author")
		}
	}

	// State transition validation
	if course.Status == entities.ContentStatusInReview {
		return nil, errors.New("course is already in review queue")
	}
	if course.Status == entities.ContentStatusPublished {
		return nil, errors.New("course is already published")
	}
	if course.Status != entities.ContentStatusDraft {
		return nil, fmt.Errorf("cannot submit course with status '%s' for review", course.Status)
	}

	course.Status = entities.ContentStatusInReview
	course.UpdatedAt = time.Now()

	if err := h.courseRepo.Update(ctx, course); err != nil {
		return nil, fmt.Errorf("failed to submit course for review: %w", err)
	}

	return course, nil
}
