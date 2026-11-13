package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

var ErrReportReasonRequired = errors.New("el motivo del reporte es obligatorio")

// CreateContentReportInput contains fields submitted by a student from mobile/web.
type CreateContentReportInput struct {
	ReporterUserID string
	LessonID       *string
	GameQuestionID *string
	Reason         string
	Description    *string
}

// CreateContentReportHandler handles creating a new student content report.
type CreateContentReportHandler struct {
	repo repositories.AuditRepository
}

// NewCreateContentReportHandler creates a CreateContentReportHandler.
func NewCreateContentReportHandler(repo repositories.AuditRepository) *CreateContentReportHandler {
	return &CreateContentReportHandler{repo: repo}
}

// Handle persists a new content report in MySQL.
func (h *CreateContentReportHandler) Handle(ctx context.Context, input CreateContentReportInput) (*entities.ContentReport, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, ErrReportReasonRequired
	}
	var reporterPtr *string
	if strings.TrimSpace(input.ReporterUserID) != "" {
		reporterPtr = &input.ReporterUserID
	}
	report := &entities.ContentReport{
		ReporterUserID: reporterPtr,
		LessonID:       input.LessonID,
		GameQuestionID: input.GameQuestionID,
		Reason:         reason,
		Description:    input.Description,
		Status:         "open",
	}
	if err := h.repo.CreateContentReport(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}
