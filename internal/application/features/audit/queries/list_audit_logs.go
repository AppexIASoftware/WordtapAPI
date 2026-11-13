package queries

import (
	"context"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// ListAuditLogsResult groups audit logs, pagination total, and KPI summary.
type ListAuditLogsResult struct {
	Summary repositories.AuditSummary `json:"summary"`
	Items   []entities.AuditLog       `json:"items"`
	Total   int64                     `json:"total"`
}

// ListAuditLogsHandler handles fetching admin audit logs and real-time summary.
type ListAuditLogsHandler struct {
	repo repositories.AuditRepository
}

// NewListAuditLogsHandler creates a ListAuditLogsHandler.
func NewListAuditLogsHandler(repo repositories.AuditRepository) *ListAuditLogsHandler {
	return &ListAuditLogsHandler{repo: repo}
}

// Handle executes the audit log query.
func (h *ListAuditLogsHandler) Handle(ctx context.Context, filter repositories.AuditLogFilter) (ListAuditLogsResult, error) {
	items, total, err := h.repo.ListAuditLogs(ctx, filter)
	if err != nil {
		return ListAuditLogsResult{}, err
	}
	summary, err := h.repo.GetAuditSummary(ctx)
	if err != nil {
		return ListAuditLogsResult{}, err
	}
	return ListAuditLogsResult{
		Summary: summary,
		Items:   items,
		Total:   total,
	}, nil
}
