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
	reviewRepo repositories.CourseReviewRepository
}

func NewSubmitCourseReviewHandler(courseRepo repositories.CourseRepository, reviewRepo repositories.CourseReviewRepository) *SubmitCourseReviewHandler {
	return &SubmitCourseReviewHandler{courseRepo: courseRepo, reviewRepo: reviewRepo}
}

func (h *SubmitCourseReviewHandler) Handle(ctx context.Context, cmd SubmitCourseReviewCommand) (*entities.Course, error) {
	if cmd.CourseID == "" {
		return nil, errors.New("course id is required")
	}

	course, err := h.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	if cmd.RequesterRole != entities.RoleInstructor || cmd.RequesterID == "" || course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
		return nil, errors.New("forbidden: only the owning teacher can submit a course for review")
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

	request := &entities.CourseReviewRequest{CourseID: course.ID, InstructorID: cmd.RequesterID, Status: entities.CourseReviewStatusPending, SubmittedAt: time.Now()}
	if err := h.reviewRepo.CreateSubmission(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to submit course for review: %w", err)
	}

	course.Status = entities.ContentStatusInReview
	course.UpdatedAt = request.SubmittedAt
	return course, nil
}
