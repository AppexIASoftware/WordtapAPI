package commands

import (
	"context"
	"errors"
	"fmt"

	courseAccess "github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
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

	// Deleting follows the same ownership rule as editing: admins delete only
	// platform courses and instructors only their own teacher courses.
	if !courseAccess.CanEditCourseContent(course, cmd.RequesterID, cmd.RequesterRole) {
		if courseAccess.EffectiveOwnerType(course) == entities.OwnerTypePlatform {
			return apperrors.New("No puedes eliminar un curso de la plataforma.")
		}
		return apperrors.New("No puedes eliminar un curso creado por un docente.")
	}

	return h.courseRepo.Delete(ctx, cmd.CourseID)
}
