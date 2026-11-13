package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ListContentReportsResult groups student content reports and total count.
type ListContentReportsResult struct {
	Items []entities.ContentReport `json:"items"`
	Total int64                    `json:"total"`
}

// ListContentReportsHandler handles fetching student content reports.
type ListContentReportsHandler struct {
	repo repositories.AuditRepository
}

// NewListContentReportsHandler creates a ListContentReportsHandler.
func NewListContentReportsHandler(repo repositories.AuditRepository) *ListContentReportsHandler {
	return &ListContentReportsHandler{repo: repo}
}

// Handle executes the content reports query.
func (h *ListContentReportsHandler) Handle(ctx context.Context, status string, limit, offset int) (ListContentReportsResult, error) {
	items, total, err := h.repo.ListContentReports(ctx, status, limit, offset)
	if err != nil {
		return ListContentReportsResult{}, err
	}
	return ListContentReportsResult{
		Items: items,
		Total: total,
	}, nil
}
