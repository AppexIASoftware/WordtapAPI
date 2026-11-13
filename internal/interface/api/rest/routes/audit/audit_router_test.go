package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	auditCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/audit/commands"
	auditQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/audit/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type stubAuditRepo struct {
	logs    []entities.AuditLog
	reports []entities.ContentReport
}

func (s *stubAuditRepo) CreateAuditLog(_ context.Context, log *entities.AuditLog) error {
	s.logs = append(s.logs, *log)
	return nil
}

func (s *stubAuditRepo) ListAuditLogs(_ context.Context, filter repositories.AuditLogFilter) ([]entities.AuditLog, int64, error) {
	return s.logs, int64(len(s.logs)), nil
}

func (s *stubAuditRepo) ListContentReports(_ context.Context, status string, limit, offset int) ([]entities.ContentReport, int64, error) {
	return s.reports, int64(len(s.reports)), nil
}

func (s *stubAuditRepo) GetContentReportByID(_ context.Context, id string) (*entities.ContentReport, error) {
	for i := range s.reports {
		if s.reports[i].ID == id {
			return &s.reports[i], nil
		}
	}
	return nil, echo.ErrNotFound
}

func (s *stubAuditRepo) UpdateContentReportStatus(_ context.Context, id, status, resolvedBy string, resolvedAt *time.Time) (*entities.ContentReport, error) {
	for i := range s.reports {
		if s.reports[i].ID == id {
			s.reports[i].Status = status
			s.reports[i].ResolvedBy = &resolvedBy
			s.reports[i].ResolvedAt = resolvedAt
			return &s.reports[i], nil
		}
	}
	return nil, echo.ErrNotFound
}

func (s *stubAuditRepo) CreateContentReport(_ context.Context, report *entities.ContentReport) error {
	if report.ID == "" {
		report.ID = "rep-new"
	}
	s.reports = append(s.reports, *report)
	return nil
}

func (s *stubAuditRepo) GetAuditSummary(_ context.Context) (repositories.AuditSummary, error) {
	return repositories.AuditSummary{
		TotalEvents:  int64(len(s.logs)),
		TotalReports: int64(len(s.reports)),
	}, nil
}

func TestAuditRouterEndpoints(t *testing.T) {
	repo := &stubAuditRepo{
		logs: []entities.AuditLog{{ID: "log-1", Action: "COURSE_APPROVE", EntityType: "course", Changes: "{}"}},
		reports: []entities.ContentReport{{ID: "rep-1", Reason: "Pronunciación incorrecta", Status: "open"}},
	}

	e := echo.New()
	v1 := e.Group("/api/v1")
	allow := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "admin-1")
			return next(c)
		}
	}

	r := NewAuditRouter(
		auditQuery.NewListAuditLogsHandler(repo),
		auditQuery.NewListContentReportsHandler(repo),
		auditCmd.NewDecideContentReportHandler(repo),
		auditCmd.NewCreateContentReportHandler(repo),
	)
	r.RegisterRoutes(v1, allow, allow)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for audit-logs, got %d: %s", rec.Code, rec.Body.String())
	}

	reqDecide := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content-reports/rep-1/decision", strings.NewReader(`{"status":"resolved"}`))
	reqDecide.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recDecide := httptest.NewRecorder()
	e.ServeHTTP(recDecide, reqDecide)
	if recDecide.Code != http.StatusOK {
		t.Fatalf("expected 200 for report decision, got %d: %s", recDecide.Code, recDecide.Body.String())
	}
	var decided entities.ContentReport
	if err := json.Unmarshal(recDecide.Body.Bytes(), &decided); err != nil || decided.Status != "resolved" {
		t.Fatalf("expected resolved report, got %+v, err=%v", decided, err)
	}
	if len(repo.logs) != 2 {
		t.Fatalf("expected audit log created on report decision, got %d logs", len(repo.logs))
	}
}
