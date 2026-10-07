package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type DeleteCourseCommand struct {
	CourseID      string
	RequesterID   string
	RequesterRole entities.UserRole
}

type DeleteCourseHandler struct {
	courseRepo repositories.CourseRepository
}

func NewDeleteCourseHandler(courseRepo repositories.CourseRepository) *DeleteCourseHandler {
	return &DeleteCourseHandler{courseRepo: courseRepo}
}

func (h *DeleteCourseHandler) Handle(ctx context.Context, cmd DeleteCourseCommand) error {
	if cmd.CourseID == "" {
		return errors.New("course id is required")
	}

	course, err := h.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return fmt.Errorf("course not found: %w", err)
	}

	if cmd.RequesterRole != entities.RoleAdmin && cmd.RequesterRole != entities.RoleModerator {
		if course.CreatedBy == nil || *course.CreatedBy != cmd.RequesterID {
			return errors.New("forbidden: only course author, admin or moderator can delete this course")
		}
	}

	return h.courseRepo.Delete(ctx, cmd.CourseID)
}
