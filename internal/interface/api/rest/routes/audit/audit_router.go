package audit

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	auditCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/audit/commands"
	auditQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/audit/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// AuditRouter exposes admin audit log and content report triage endpoints.
type AuditRouter struct {
	listLogs      *auditQuery.ListAuditLogsHandler
	listReports   *auditQuery.ListContentReportsHandler
	decideReport  *auditCmd.DecideContentReportHandler
	createReport  *auditCmd.CreateContentReportHandler
}

// NewAuditRouter creates a new AuditRouter.
func NewAuditRouter(
	listLogs *auditQuery.ListAuditLogsHandler,
	listReports *auditQuery.ListContentReportsHandler,
	decideReport *auditCmd.DecideContentReportHandler,
	createReport *auditCmd.CreateContentReportHandler,
) *AuditRouter {
	return &AuditRouter{
		listLogs:     listLogs,
		listReports:  listReports,
		decideReport: decideReport,
		createReport: createReport,
	}
}

// RegisterRoutes mounts audit & content report endpoints on v1.
func (r *AuditRouter) RegisterRoutes(v1 *echo.Group, authRequired, adminOrModerator echo.MiddlewareFunc) {
	v1.GET("/admin/audit-logs", r.ListLogs, authRequired, adminOrModerator)
	v1.GET("/admin/content-reports", r.ListReports, authRequired, adminOrModerator)
	v1.POST("/admin/content-reports/:id/decision", r.DecideReport, authRequired, adminOrModerator)
	v1.POST("/content-reports", r.CreateReport, authRequired)
}

func (r *AuditRouter) ListLogs(c *echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	res, err := r.listLogs.Handle(c.Request().Context(), repositories.AuditLogFilter{
		Category: c.QueryParam("cat"),
		Query:    c.QueryParam("q"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la bitácora de auditoría.", err)
	}
	return c.JSON(http.StatusOK, res)
}

func (r *AuditRouter) ListReports(c *echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	res, err := r.listReports.Handle(c.Request().Context(), c.QueryParam("status"), limit, offset)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar los reportes de contenido.", err)
	}
	return c.JSON(http.StatusOK, res)
}

type decideReportBody struct {
	Status string `json:"status"`
}

func (r *AuditRouter) DecideReport(c *echo.Context) error {
	var body decideReportBody
	if err := c.Bind(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}
	userID, _ := c.Get("user_id").(string)
	updated, err := r.decideReport.Handle(c.Request().Context(), auditCmd.DecideContentReportInput{
		ReportID:    c.Param("id"),
		Status:      body.Status,
		ActorUserID: userID,
		IPAddress:   c.RealIP(),
	})
	if err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, auditCmd.ErrReportNotFound) {
			statusCode = http.StatusNotFound
		}
		return response.Error(c, statusCode, err.Error(), err)
	}
	return c.JSON(http.StatusOK, updated)
}

type createReportBody struct {
	LessonID       *string `json:"lesson_id"`
	GameQuestionID *string `json:"game_question_id"`
	Reason         string  `json:"reason"`
	Description    *string `json:"description"`
}

func (r *AuditRouter) CreateReport(c *echo.Context) error {
	var body createReportBody
	if err := c.Bind(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}
	userID, _ := c.Get("user_id").(string)
	created, err := r.createReport.Handle(c.Request().Context(), auditCmd.CreateContentReportInput{
		ReporterUserID: userID,
		LessonID:       body.LessonID,
		GameQuestionID: body.GameQuestionID,
		Reason:         body.Reason,
		Description:    body.Description,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error(), err)
	}
	return c.JSON(http.StatusCreated, created)
}
