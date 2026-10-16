package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
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
		return nil, apperrors.New("No tienes permisos para ver las revisiones pendientes.")
	}
	return h.repo.ListPending(ctx)
}
