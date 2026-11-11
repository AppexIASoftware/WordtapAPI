package classroom

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type NotificationRouter struct {
	handler *queries.ListClassroomNotificationsHandler
}

func NewNotificationRouter(handler *queries.ListClassroomNotificationsHandler) *NotificationRouter {
	return &NotificationRouter{handler: handler}
}

func (r *NotificationRouter) RegisterRoutes(v1 *echo.Group, authRequired, instructorOrAdmin echo.MiddlewareFunc) {
	group := v1.Group("/teacher/classrooms")
	group.Use(authRequired, instructorOrAdmin)
	group.GET("/:id/notifications", r.List)
}

func (r *NotificationRouter) List(c *echo.Context) error {
	teacherID, ok := c.Get("user_id").(string)
	if !ok || teacherID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Autenticación requerida.")
	}

	classroomID := c.Param("id")
	if classroomID == "" {
		return response.Fail(c, http.StatusBadRequest, "ID de aula requerido.")
	}

	// Pagination parameters
	limit := 20
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Optional query filters
	var statusEnum *entities.ClassroomNotificationStatus
	if s := c.QueryParam("status"); s != "" {
		status := entities.ClassroomNotificationStatus(s)
		statusEnum = &status
	}
	var dateFrom *time.Time
	if df := c.QueryParam("date_from"); df != "" {
		if parsed, err := time.Parse(time.RFC3339, df); err == nil {
			dateFrom = &parsed
		}
	}
	var dateTo *time.Time
	if dt := c.QueryParam("date_to"); dt != "" {
		if parsed, err := time.Parse(time.RFC3339, dt); err == nil {
			dateTo = &parsed
		}
	}

	query := queries.ListClassroomNotificationsQuery{
		Scope:       queries.NotificationScopeTeacher,
		ClassroomID: classroomID,
		TeacherID:   teacherID,
		Limit:       limit,
		Offset:      offset,
		Status:      statusEnum,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
	}

	result, err := r.handler.Handle(c.Request().Context(), query)
	if err != nil {
		if err.Error() == "aula no encontrada" {
			return response.Fail(c, http.StatusNotFound, err.Error())
		}
		if err.Error() == "no tienes permisos para ver el historial de esta aula" {
			return response.Fail(c, http.StatusForbidden, err.Error())
		}
		return response.Fail(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"notifications": result.Notifications,
		"pagination": map[string]any{
			"total":  result.Total,
			"limit":  result.Limit,
			"offset": result.Offset,
		},
	})
}