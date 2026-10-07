package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrCourseAccessDenied = errors.New("access denied: course is not published")
)

type GetCourseDetailQuery struct {
	CourseID      string
	RequesterID   string
	RequesterRole entities.UserRole
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

	course, err := h.courseRepo.FindByID(ctx, query.CourseID)
	if err != nil || course == nil {
		return nil, ErrCourseNotFound
	}

	if course.Status != entities.ContentStatusPublished {
		isOwner := query.RequesterID != "" && course.CreatedBy != nil && *course.CreatedBy == query.RequesterID
		isAdmin := query.RequesterRole == entities.RoleAdmin
		isModerator := query.RequesterRole == entities.RoleModerator && course.Status == entities.ContentStatusInReview
		if !isOwner && !isAdmin && !isModerator {
			return nil, ErrCourseAccessDenied
		}
	}

	return course, nil
}
