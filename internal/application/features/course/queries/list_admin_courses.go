package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListAdminCoursesQuery struct{}

type ListAdminCoursesHandler struct {
	courseRepo repositories.CourseRepository
}

func NewListAdminCoursesHandler(courseRepo repositories.CourseRepository) *ListAdminCoursesHandler {
	return &ListAdminCoursesHandler{courseRepo: courseRepo}
}

func (h *ListAdminCoursesHandler) Handle(ctx context.Context, _ ListAdminCoursesQuery) ([]entities.Course, error) {
	return h.courseRepo.FindAll(ctx)
}
