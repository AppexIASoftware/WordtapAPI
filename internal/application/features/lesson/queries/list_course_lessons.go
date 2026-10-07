package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrCourseLessonsNotFound     = errors.New("course not found")
	ErrCourseLessonsAccessDenied = errors.New("access denied: course is not published")
)

type ListCourseLessonsQuery struct {
	CourseID      string
	RequesterID   string
	RequesterRole entities.UserRole
}

type ListCourseLessonsHandler struct {
	lessonRepo repositories.LessonRepository
	courseRepo repositories.CourseRepository
}

func NewListCourseLessonsHandler(lessonRepo repositories.LessonRepository, courseRepo repositories.CourseRepository) *ListCourseLessonsHandler {
	return &ListCourseLessonsHandler{lessonRepo: lessonRepo, courseRepo: courseRepo}
}

func (h *ListCourseLessonsHandler) Handle(ctx context.Context, query ListCourseLessonsQuery) ([]entities.Lesson, error) {
	if query.CourseID == "" {
		return nil, errors.New("course id is required")
	}

	if h.courseRepo != nil {
		course, err := h.courseRepo.FindByID(ctx, query.CourseID)
		if err != nil || course == nil {
			return nil, ErrCourseLessonsNotFound
		}

		isOwner := query.RequesterID != "" && course.CreatedBy != nil && *course.CreatedBy == query.RequesterID
		isAdmin := query.RequesterRole == entities.RoleAdmin
		isModerator := query.RequesterRole == entities.RoleModerator && course.Status == entities.ContentStatusInReview

		if course.Status != entities.ContentStatusPublished && !isOwner && !isAdmin && !isModerator {
			return nil, ErrCourseLessonsAccessDenied
		}

		lessons, err := h.lessonRepo.FindByCourseID(ctx, query.CourseID)
		if err != nil {
			return nil, err
		}

		if !isOwner && !isAdmin && !isModerator {
			var publishedLessons []entities.Lesson
			for _, l := range lessons {
				if l.Status == entities.ContentStatusPublished {
					l.Items = nil
					publishedLessons = append(publishedLessons, l)
				}
			}
			return publishedLessons, nil
		}

		return lessons, nil
	}

	return h.lessonRepo.FindByCourseID(ctx, query.CourseID)
}
