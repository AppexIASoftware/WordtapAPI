package repositories

import (
	"context"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

// AuditLogFilter defines query filters for admin audit logs.
type AuditLogFilter struct {
	Category string
	Query    string
	Limit    int
	Offset   int
}

// AuditSummary holds real-time counts for the admin audit dashboard.
type AuditSummary struct {
	EventsToday          int64 `json:"events_today"`
	OpenReports          int64 `json:"open_reports"`
	ReviewingReports     int64 `json:"reviewing_reports"`
	ResolvedTodayReports int64 `json:"resolved_today_reports"`
	ApprovalsThisMonth   int64 `json:"approvals_this_month"`
	DistinctIPs          int64 `json:"distinct_ips"`
	TotalEvents          int64 `json:"total_events"`
	TotalReports         int64 `json:"total_reports"`
}

// AuditRepository defines persistence operations for audit logs and content reports.
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, log *entities.AuditLog) error
	ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]entities.AuditLog, int64, error)
	ListContentReports(ctx context.Context, status string, limit, offset int) ([]entities.ContentReport, int64, error)
	GetContentReportByID(ctx context.Context, id string) (*entities.ContentReport, error)
	UpdateContentReportStatus(ctx context.Context, id, status, resolvedBy string, resolvedAt *time.Time) (*entities.ContentReport, error)
	CreateContentReport(ctx context.Context, report *entities.ContentReport) error
	GetAuditSummary(ctx context.Context) (AuditSummary, error)
}
