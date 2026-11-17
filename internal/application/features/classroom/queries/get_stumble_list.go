package queries

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrClassroomNotFound  = errors.New("aula no encontrada")
	ErrClassroomForbidden = errors.New("no tienes permisos para consultar los tropiezos de esta aula")
)

type GetStumbleListQuery struct {
	ClassroomID   string
	RequesterID   string
	RequesterRole entities.UserRole
}

type GetStumbleListHandler struct {
	classroomRepo repositories.ClassroomRepository
}

func NewGetStumbleListHandler(classroomRepo repositories.ClassroomRepository) *GetStumbleListHandler {
	return &GetStumbleListHandler{classroomRepo: classroomRepo}
}

func (h *GetStumbleListHandler) Handle(ctx context.Context, query GetStumbleListQuery) ([]entities.StudentStumble, error) {
	clsID := strings.TrimSpace(query.ClassroomID)
	if clsID == "" {
		return nil, errors.New("el identificador del aula es obligatorio")
	}

	cls, err := h.classroomRepo.FindByID(ctx, clsID)
	if err != nil || cls == nil {
		return nil, ErrClassroomNotFound
	}

	role := entities.UserRole(strings.ToLower(strings.TrimSpace(string(query.RequesterRole))))
	if role != entities.RoleAdmin && role != entities.RoleModerator {
		if strings.TrimSpace(query.RequesterID) == "" || cls.TeacherID != strings.TrimSpace(query.RequesterID) {
			return nil, ErrClassroomForbidden
		}
	}

	stumbles, err := h.classroomRepo.ListStumbles(ctx, clsID)
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
