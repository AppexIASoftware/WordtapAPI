package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListCourseReviewsQuery struct {
	RequesterID   string
	RequesterRole entities.UserRole
}
type ListCourseReviewsHandler struct {
	repo repositories.CourseReviewRepository
}

func NewListCourseReviewsHandler(repo repositories.CourseReviewRepository) *ListCourseReviewsHandler {
	return &ListCourseReviewsHandler{repo: repo}
}
func (h *ListCourseReviewsHandler) Handle(ctx context.Context, q ListCourseReviewsQuery) ([]entities.CourseReviewRequest, error) {
	if (q.RequesterRole != entities.RoleAdmin && q.RequesterRole != entities.RoleModerator) || q.RequesterID == "" {
		return nil, errors.New("forbidden: admin or moderator role required")
	}
	return h.repo.ListPending(ctx)
}
