package commands

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ReviewCourseCommand struct {
	ReviewID      string
	ReviewerID    string
	ReviewerRole  entities.UserRole
	Status        entities.CourseReviewStatus
	FeedbackNotes *string
}

type ReviewCourseHandler struct {
	reviewRepo repositories.CourseReviewRepository
}

func NewReviewCourseHandler(repo repositories.CourseReviewRepository) *ReviewCourseHandler {
	return &ReviewCourseHandler{reviewRepo: repo}
}

func (h *ReviewCourseHandler) Handle(ctx context.Context, cmd ReviewCourseCommand) (*entities.CourseReviewRequest, error) {
	if cmd.ReviewerRole != entities.RoleAdmin || cmd.ReviewerID == "" {
		return nil, errors.New("forbidden: admin role required")
	}
	if cmd.ReviewID == "" {
		return nil, errors.New("review id is required")
	}
	if cmd.Status != entities.CourseReviewStatusApproved && cmd.Status != entities.CourseReviewStatusChangesRequested && cmd.Status != entities.CourseReviewStatusRejected {
		return nil, errors.New("invalid review decision")
	}
	return h.reviewRepo.Decide(ctx, cmd.ReviewID, cmd.ReviewerID, cmd.Status, cmd.FeedbackNotes)
}
