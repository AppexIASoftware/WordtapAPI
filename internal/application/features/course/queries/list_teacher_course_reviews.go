package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListTeacherCourseReviewsQuery struct {
	InstructorID  string
	RequesterID   string
	RequesterRole entities.UserRole
}
type ListTeacherCourseReviewsHandler struct {
	repo repositories.CourseReviewRepository
}

func NewListTeacherCourseReviewsHandler(repo repositories.CourseReviewRepository) *ListTeacherCourseReviewsHandler {
	return &ListTeacherCourseReviewsHandler{repo: repo}
}
func (h *ListTeacherCourseReviewsHandler) Handle(ctx context.Context, q ListTeacherCourseReviewsQuery) ([]entities.CourseReviewRequest, error) {
	if q.RequesterRole != entities.RoleInstructor || q.RequesterID == "" || q.InstructorID != q.RequesterID {
		return nil, errors.New("forbidden: teachers can only read their own review history")
	}
	return h.repo.ListByInstructor(ctx, q.InstructorID)
}
