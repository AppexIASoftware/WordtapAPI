package notification

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type ClassroomNotificationRouter struct {
	handler *queries.ListClassroomNotificationsHandler
}

func NewClassroomNotificationRouter(handler *queries.ListClassroomNotificationsHandler) *ClassroomNotificationRouter {
	return &ClassroomNotificationRouter{handler: handler}
}

func (r *ClassroomNotificationRouter) RegisterRoutes(v1 *echo.Group, authRequired, adminOrModerator echo.MiddlewareFunc) {
	group := v1.Group("/admin/notifications")
	group.Use(authRequired, adminOrModerator)
	group.GET("", r.List)
}

func (r *ClassroomNotificationRouter) List(c *echo.Context) error {
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
	var classroomID, teacherID, courseID, lessonID string
	if v := c.QueryParam("classroom_id"); v != "" {
		classroomID = v
	}
	if v := c.QueryParam("teacher_id"); v != "" {
		teacherID = v
	}
	if v := c.QueryParam("course_id"); v != "" {
		courseID = v
	}
	if v := c.QueryParam("lesson_id"); v != "" {
		lessonID = v
	}

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
		Scope:       queries.NotificationScopeAdmin,
		ClassroomID: classroomID,
		TeacherID:   teacherID,
		CourseID:    courseID,
		LessonID:    lessonID,
		Limit:       limit,
		Offset:      offset,
		Status:      statusEnum,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
	}

	result, err := r.handler.Handle(c.Request().Context(), query)
	if err != nil {
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