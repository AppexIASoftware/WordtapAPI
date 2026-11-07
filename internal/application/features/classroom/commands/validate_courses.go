package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type CourseLookup interface {
	FindByID(context.Context, string) (*entities.Course, error)
}

func validateCourses(ctx context.Context, repo CourseLookup, teacherID string, ids []string) error {
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if repo == nil {
			return errors.New("course lookup is required")
		}
		course, err := repo.FindByID(ctx, id)
		if err != nil || course == nil || course.Status != entities.ContentStatusPublished {
			return apperrors.New("Solo puedes asignar cursos publicados y disponibles.")
		}
		if course.OwnerType != entities.OwnerTypePlatform && (course.CreatedBy == nil || *course.CreatedBy != teacherID) {
			return apperrors.New("No tienes permisos para asignar uno de los cursos seleccionados.")
		}
	}
	return nil
}
