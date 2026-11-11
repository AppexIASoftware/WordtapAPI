package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ListAdminCoursesQuery optionally narrows the admin listing by publication status and owner type.
type ListAdminCoursesQuery struct {
	Status    entities.ContentStatus
	OwnerType entities.OwnerType
}

type ListAdminCoursesHandler struct {
	courseRepo repositories.CourseRepository
}

func NewListAdminCoursesHandler(courseRepo repositories.CourseRepository) *ListAdminCoursesHandler {
	return &ListAdminCoursesHandler{courseRepo: courseRepo}
}

func (h *ListAdminCoursesHandler) Handle(ctx context.Context, query ListAdminCoursesQuery) ([]entities.Course, error) {
	if query.Status != "" || query.OwnerType != "" {
		return h.courseRepo.FindAllByStatus(ctx, query.Status, query.OwnerType)
	}
	return h.courseRepo.FindAll(ctx)
}
