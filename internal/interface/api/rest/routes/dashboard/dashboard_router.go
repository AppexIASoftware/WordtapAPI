package dashboard

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	dashQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/dashboard/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// DashboardRouter exposes executive governance dashboard endpoints.
type DashboardRouter struct {
	getAdminDashboard   *dashQuery.GetAdminDashboardHandler
	getTeacherDashboard *dashQuery.GetTeacherDashboardHandler
	listGlobalStudents  *dashQuery.ListGlobalStudentsHandler
}

// NewDashboardRouter creates a DashboardRouter.
func NewDashboardRouter(getAdminDashboard *dashQuery.GetAdminDashboardHandler) *DashboardRouter {
	return &DashboardRouter{getAdminDashboard: getAdminDashboard}
}

// WithTeacherHandler attaches the teacher dashboard query handler.
func (r *DashboardRouter) WithTeacherHandler(h *dashQuery.GetTeacherDashboardHandler) *DashboardRouter {
	r.getTeacherDashboard = h
	return r
}

// WithGlobalStudentsHandler attaches the global students telemetry query handler.
func (r *DashboardRouter) WithGlobalStudentsHandler(h *dashQuery.ListGlobalStudentsHandler) *DashboardRouter {
	r.listGlobalStudents = h
	return r
}

// RegisterRoutes registers dashboard routes on v1.
func (r *DashboardRouter) RegisterRoutes(v1 *echo.Group, authRequired, adminOrModerator echo.MiddlewareFunc) {
	v1.GET("/admin/dashboard", r.GetAdminDashboard, authRequired, adminOrModerator)
}

// RegisterTeacherRoutes registers teacher dashboard metrics route on v1.
func (r *DashboardRouter) RegisterTeacherRoutes(v1 *echo.Group, authRequired, instructorOrAdmin echo.MiddlewareFunc) {
	if r.getTeacherDashboard != nil {
		v1.GET("/teacher/dashboard", r.GetTeacherDashboard, authRequired, instructorOrAdmin)
	}
	if r.listGlobalStudents != nil {
		v1.GET("/teacher/global-students", r.ListGlobalStudents, authRequired, instructorOrAdmin)
	}
}

// ListGlobalStudents returns filtered global student telemetry from MySQL.
func (r *DashboardRouter) ListGlobalStudents(c *echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	data, err := r.listGlobalStudents.Handle(c.Request().Context(), dashQuery.GlobalStudentFilter{
		Query:    c.QueryParam("q"),
		Plan:     c.QueryParam("plan"),
		Level:    c.QueryParam("level"),
		CourseID: c.QueryParam("course_id"),
		Status:   c.QueryParam("status"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el listado global de estudiantes.", err)
	}
	return c.JSON(http.StatusOK, data)
}

// GetTeacherDashboard returns real-time learning, vault, and revenue KPIs from MySQL.
func (r *DashboardRouter) GetTeacherDashboard(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	period := c.QueryParam("period")
	data, err := r.getTeacherDashboard.Handle(c.Request().Context(), userID, period)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el dashboard docente.", err)
	}
	return c.JSON(http.StatusOK, data)
}

// GetAdminDashboard returns real-time platform metrics and recent activity from MySQL.
func (r *DashboardRouter) GetAdminDashboard(c *echo.Context) error {
	data, err := r.getAdminDashboard.Handle(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el dashboard administrativo.", err)
	}
	return c.JSON(http.StatusOK, data)
}
