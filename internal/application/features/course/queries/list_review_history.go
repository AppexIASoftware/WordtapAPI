package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListReviewHistoryHandler struct {
	repo repositories.CourseReviewHistoryRepository
}

func NewListReviewHistoryHandler(repo repositories.CourseReviewHistoryRepository) *ListReviewHistoryHandler {
	return &ListReviewHistoryHandler{repo: repo}
}

func (h *ListReviewHistoryHandler) Handle(ctx context.Context, q ListCourseReviewsQuery) ([]entities.CourseReviewRequest, error) {
	if q.RequesterID == "" || (q.RequesterRole != entities.RoleAdmin && q.RequesterRole != entities.RoleModerator) {
		return nil, apperrors.New("No tienes permisos para ver el historial de revisiones.")
	}
	return h.repo.ListDecided(ctx)
}
