package queries

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type GetStumbleListQuery struct {
	ClassroomID string
}

type GetStumbleListHandler struct {
	classroomRepo repositories.ClassroomRepository
}

func NewGetStumbleListHandler(classroomRepo repositories.ClassroomRepository) *GetStumbleListHandler {
	return &GetStumbleListHandler{classroomRepo: classroomRepo}
}

func (h *GetStumbleListHandler) Handle(ctx context.Context, query GetStumbleListQuery) ([]entities.StudentStumble, error) {
	if query.ClassroomID == "" {
		return nil, errors.New("el identificador del aula es obligatorio")
	}

	stumbles, err := h.classroomRepo.ListStumbles(ctx, query.ClassroomID)
	if err != nil {
		return nil, err
	}

	filtered := make([]entities.StudentStumble, 0, len(stumbles))
	for _, s := range stumbles {
		if s.FailRate >= 40 {
			filtered = append(filtered, s)
		}
	}

	return filtered, nil
}
