package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListCourseLessonsQuery struct {
	CourseID string
}

type ListCourseLessonsHandler struct {
	lessonRepo repositories.LessonRepository
}

func NewListCourseLessonsHandler(lessonRepo repositories.LessonRepository) *ListCourseLessonsHandler {
	return &ListCourseLessonsHandler{lessonRepo: lessonRepo}
}

func (h *ListCourseLessonsHandler) Handle(ctx context.Context, query ListCourseLessonsQuery) ([]entities.Lesson, error) {
	if query.CourseID == "" {
		return nil, errors.New("course id is required")
	}

	return h.lessonRepo.FindByCourseID(ctx, query.CourseID)
}
