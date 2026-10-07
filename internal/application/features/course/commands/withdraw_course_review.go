package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type WithdrawCourseReviewCommand struct {
	ID            string // Review request ID or Course ID
	RequesterID   string
	RequesterRole entities.UserRole
}

type WithdrawCourseReviewHandler struct {
	reviewRepo repositories.CourseReviewRepository
}

func NewWithdrawCourseReviewHandler(reviewRepo repositories.CourseReviewRepository) *WithdrawCourseReviewHandler {
	return &WithdrawCourseReviewHandler{reviewRepo: reviewRepo}
}

func (h *WithdrawCourseReviewHandler) Handle(ctx context.Context, cmd WithdrawCourseReviewCommand) (*entities.CourseReviewRequest, error) {
	if cmd.ID == "" {
		return nil, errors.New("id is required")
	}
	if cmd.RequesterRole != entities.RoleInstructor || cmd.RequesterID == "" {
		return nil, errors.New("forbidden: only instructor can withdraw a course review request")
	}

	request, err := h.reviewRepo.Withdraw(ctx, cmd.ID, cmd.RequesterID)
	if err != nil {
		return nil, fmt.Errorf("failed to withdraw review: %w", err)
	}
	return request, nil
}
