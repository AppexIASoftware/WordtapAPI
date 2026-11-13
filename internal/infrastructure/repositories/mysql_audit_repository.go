package repositories

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domain "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// MySQLAuditRepository implements domain.AuditRepository using GORM/MySQL.
type MySQLAuditRepository struct {
	db *gorm.DB
}

// NewMySQLAuditRepository creates a new MySQLAuditRepository.
func NewMySQLAuditRepository(db *gorm.DB) *MySQLAuditRepository {
	return &MySQLAuditRepository{db: db}
}

// CreateAuditLog inserts an immutable audit log record.
func (r *MySQLAuditRepository) CreateAuditLog(ctx context.Context, log *entities.AuditLog) error {
	if log == nil {
		return nil
	}
	if strings.TrimSpace(log.Changes) == "" {
		log.Changes = "{}"
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// ListAuditLogs returns filtered audit logs ordered newest first.
func (r *MySQLAuditRepository) ListAuditLogs(ctx context.Context, filter domain.AuditLogFilter) ([]entities.AuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.AuditLog{})

	switch strings.TrimSpace(filter.Category) {
	case "review":
		q = q.Where("entity_type IN ? OR action LIKE ?", []string{"course", "course_review", "content_report"}, "COURSE_%")
	case "staff":
		q = q.Where("entity_type IN ? OR action LIKE ? OR action LIKE ?", []string{"user", "staff", "admin_role"}, "STAFF_%", "ROLE_%")
	case "admob":
		q = q.Where("entity_type IN ? OR action LIKE ? OR action LIKE ?", []string{"ad_config", "platform_setting", "monetization"}, "ADMOB_%", "SETTING_%")
	case "vault":
		q = q.Where("entity_type IN ? OR action LIKE ?", []string{"vault_item", "content_bank"}, "%VAULT%")
	}

	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Joins("LEFT JOIN users ON users.id = audit_logs.actor_user_id").
			Where(
				"LOWER(audit_logs.action) LIKE ? OR LOWER(audit_logs.entity_type) LIKE ? OR LOWER(COALESCE(audit_logs.entity_id, '')) LIKE ? OR LOWER(COALESCE(audit_logs.ip_address, '')) LIKE ? OR LOWER(COALESCE(users.name, '')) LIKE ? OR LOWER(COALESCE(users.email, '')) LIKE ?",
				like, like, like, like, like, like,
			)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var items []entities.AuditLog
	if err := q.Preload("ActorUser").
		Order("audit_logs.created_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListContentReports returns student content reports ordered newest first.
func (r *MySQLAuditRepository) ListContentReports(ctx context.Context, status string, limit, offset int) ([]entities.ContentReport, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.ContentReport{})
	if s := strings.TrimSpace(status); s != "" && s != "all" {
		q = q.Where("status = ?", s)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var items []entities.ContentReport
	if err := q.Preload("ReporterUser").
		Preload("Lesson").
		Preload("GameQuestion").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetContentReportByID finds a single content report by ID.
func (r *MySQLAuditRepository) GetContentReportByID(ctx context.Context, id string) (*entities.ContentReport, error) {
	var item entities.ContentReport
	if err := r.db.WithContext(ctx).
		Preload("ReporterUser").
		Preload("Lesson").
		Preload("GameQuestion").
		First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateContentReportStatus updates a report's triage status and resolution metadata.
func (r *MySQLAuditRepository) UpdateContentReportStatus(ctx context.Context, id, status, resolvedBy string, resolvedAt *time.Time) (*entities.ContentReport, error) {
	updates := map[string]any{
		"status":      status,
		"resolved_by": nil,
		"resolved_at": nil,
	}
	if resolvedBy != "" {
		updates["resolved_by"] = resolvedBy
	}
	if resolvedAt != nil {
		updates["resolved_at"] = *resolvedAt
	}
	if err := r.db.WithContext(ctx).Model(&entities.ContentReport{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetContentReportByID(ctx, id)
}

// CreateContentReport inserts a new student content report.
func (r *MySQLAuditRepository) CreateContentReport(ctx context.Context, report *entities.ContentReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// GetAuditSummary computes real-time counts from MySQL.
func (r *MySQLAuditRepository) GetAuditSummary(ctx context.Context) (domain.AuditSummary, error) {
	var summary domain.AuditSummary
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	r.db.WithContext(ctx).Model(&entities.AuditLog{}).Count(&summary.TotalEvents)
	r.db.WithContext(ctx).Model(&entities.AuditLog{}).Where("created_at >= ?", startOfDay).Count(&summary.EventsToday)
	r.db.WithContext(ctx).Model(&entities.AuditLog{}).
		Where("ip_address IS NOT NULL AND ip_address != ''").
		Distinct("ip_address").
		Count(&summary.DistinctIPs)

	r.db.WithContext(ctx).Model(&entities.ContentReport{}).Count(&summary.TotalReports)
	r.db.WithContext(ctx).Model(&entities.ContentReport{}).Where("status = ?", "open").Count(&summary.OpenReports)
	r.db.WithContext(ctx).Model(&entities.ContentReport{}).Where("status = ?", "reviewing").Count(&summary.ReviewingReports)
	r.db.WithContext(ctx).Model(&entities.ContentReport{}).
		Where("status = ? AND resolved_at >= ?", "resolved", startOfDay).
		Count(&summary.ResolvedTodayReports)

	r.db.WithContext(ctx).Model(&entities.CourseReviewRequest{}).
		Where("status = ? AND reviewed_at >= ?", entities.CourseReviewStatusApproved, startOfMonth).
		Count(&summary.ApprovalsThisMonth)

	return summary, nil
}
