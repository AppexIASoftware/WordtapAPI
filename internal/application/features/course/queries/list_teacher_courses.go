package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListTeacherCoursesQuery struct {
	AuthorID string
}

type ListTeacherCoursesHandler struct {
	courseRepo repositories.CourseRepository
}

func NewListTeacherCoursesHandler(courseRepo repositories.CourseRepository) *ListTeacherCoursesHandler {
	return &ListTeacherCoursesHandler{courseRepo: courseRepo}
}

func (h *ListTeacherCoursesHandler) Handle(ctx context.Context, query ListTeacherCoursesQuery) ([]entities.Course, error) {
	if query.AuthorID == "" {
		return nil, errors.New("author id is required")
	}

	return h.courseRepo.FindByAuthor(ctx, query.AuthorID)
}
