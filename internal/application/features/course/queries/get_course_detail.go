package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type GetCourseDetailQuery struct {
	CourseID string
}

type GetCourseDetailHandler struct {
	courseRepo repositories.CourseRepository
}

func NewGetCourseDetailHandler(courseRepo repositories.CourseRepository) *GetCourseDetailHandler {
	return &GetCourseDetailHandler{courseRepo: courseRepo}
}

func (h *GetCourseDetailHandler) Handle(ctx context.Context, query GetCourseDetailQuery) (*entities.Course, error) {
	if query.CourseID == "" {
		return nil, errors.New("course id is required")
	}

	return h.courseRepo.FindByID(ctx, query.CourseID)
}
