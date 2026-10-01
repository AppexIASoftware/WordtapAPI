package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListPublishedCoursesQuery struct{}

type ListPublishedCoursesHandler struct {
	courseRepo repositories.CourseRepository
}

func NewListPublishedCoursesHandler(courseRepo repositories.CourseRepository) *ListPublishedCoursesHandler {
	return &ListPublishedCoursesHandler{courseRepo: courseRepo}
}

func (h *ListPublishedCoursesHandler) Handle(ctx context.Context, _ ListPublishedCoursesQuery) ([]entities.Course, error) {
	return h.courseRepo.FindPublished(ctx)
}
