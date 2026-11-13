package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var (
	ErrInvalidReportStatus = errors.New("estado de reporte inválido: debe ser reviewing, resolved o dismissed")
	ErrReportNotFound      = errors.New("reporte no encontrado")
)

// DecideContentReportInput contains parameters for triaging a student content report.
type DecideContentReportInput struct {
	ReportID    string
	Status      string
	ActorUserID string
	IPAddress   string
}

// DecideContentReportHandler handles resolving, dismissing, or marking a report in review.
type DecideContentReportHandler struct {
	repo repositories.AuditRepository
}

// NewDecideContentReportHandler creates a DecideContentReportHandler.
func NewDecideContentReportHandler(repo repositories.AuditRepository) *DecideContentReportHandler {
	return &DecideContentReportHandler{repo: repo}
}

// Handle updates the content report status and records an immutable audit log entry.
func (h *DecideContentReportHandler) Handle(ctx context.Context, input DecideContentReportInput) (*entities.ContentReport, error) {
	status := strings.TrimSpace(input.Status)
	if status != "reviewing" && status != "resolved" && status != "dismissed" {
		return nil, ErrInvalidReportStatus
	}

	existing, err := h.repo.GetContentReportByID(ctx, input.ReportID)
	if err != nil || existing == nil {
		return nil, ErrReportNotFound
	}

	var resolvedBy string
	var resolvedAt *time.Time
	if status == "resolved" || status == "dismissed" {
		resolvedBy = input.ActorUserID
		now := time.Now().UTC()
		resolvedAt = &now
	}

	updated, err := h.repo.UpdateContentReportStatus(ctx, input.ReportID, status, resolvedBy, resolvedAt)
	if err != nil {
		return nil, err
	}

	changesJSON, _ := json.Marshal(map[string]any{
		"summary":    "Actualizó estado de reporte de contenido",
		"old_status": existing.Status,
		"new_status": status,
		"reason":     existing.Reason,
	})
	var actorPtr *string
	if strings.TrimSpace(input.ActorUserID) != "" {
		actorPtr = &input.ActorUserID
	}
	var ipPtr *string
	if strings.TrimSpace(input.IPAddress) != "" {
		ipPtr = &input.IPAddress
	}
	entityID := existing.ID
	_ = h.repo.CreateAuditLog(ctx, &entities.AuditLog{
		ActorUserID: actorPtr,
		Action:      "CONTENT_REPORT_" + strings.ToUpper(status),
		EntityType:  "content_report",
		EntityID:    &entityID,
		Changes:     string(changesJSON),
		IPAddress:   ipPtr,
	})

	return updated, nil
}
