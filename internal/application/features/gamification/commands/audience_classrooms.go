package commands

import (
	"context"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/apperrors"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type audienceClassroomRepository interface {
	FindByID(context.Context, string) (*entities.Classroom, error)
}

func validateAudienceClassrooms(ctx context.Context, repo audienceClassroomRepository, teacherID string, ids []string) error {
	if len(ids) == 0 || repo == nil {
		return apperrors.New("Selecciona al menos un aula válida.")
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || id != strings.TrimSpace(id) || seen[id] {
			return apperrors.New("La selección de aulas no es válida.")
		}
		seen[id] = true
		classroom, err := repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if classroom == nil || classroom.TeacherID != teacherID || !classroom.IsActive {
			return apperrors.New("Solo puedes seleccionar tus propias aulas activas.")
		}
	}
	return nil
}
