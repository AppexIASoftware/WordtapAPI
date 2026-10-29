package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type audienceCourseRepository interface {
	FindByAuthor(context.Context, string) ([]entities.Course, error)
}

func validateAudienceCourses(ctx context.Context, repo audienceCourseRepository, teacherID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if repo == nil {
		return errors.New("audience course repository is not configured")
	}
	courses, err := repo.FindByAuthor(ctx, teacherID)
	if err != nil {
		return err
	}
	owned := make(map[string]bool, len(courses))
	for _, course := range courses {
		owned[course.ID] = true
	}
	for _, id := range ids {
		if id == "" || id != strings.TrimSpace(id) || !owned[id] {
			return apperrors.New("Solo puedes asignar cursos propios a la audiencia.")
		}
	}
	return nil
}
