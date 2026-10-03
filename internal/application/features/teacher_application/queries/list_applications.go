package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type ListApplicationsQuery struct {
	Status *entities.ApplicationStatus
	Limit  int
	Offset int
}

type ListApplicationsResultDTO struct {
	Applications []entities.TeacherApplication `json:"applications"`
	Total        int64                         `json:"total"`
	Limit        int                           `json:"limit"`
	Offset       int                           `json:"offset"`
}

type ListApplicationsHandler struct {
	appRepo repositories.TeacherApplicationRepository
}

func NewListApplicationsHandler(appRepo repositories.TeacherApplicationRepository) *ListApplicationsHandler {
	return &ListApplicationsHandler{appRepo: appRepo}
}

func (h *ListApplicationsHandler) Handle(ctx context.Context, q ListApplicationsQuery) (*ListApplicationsResultDTO, error) {
	apps, total, err := h.appRepo.List(ctx, q.Status, q.Limit, q.Offset)
	if err != nil {
		return nil, err
	}

	return &ListApplicationsResultDTO{
		Applications: apps,
		Total:        total,
		Limit:        q.Limit,
		Offset:       q.Offset,
	}, nil
}
