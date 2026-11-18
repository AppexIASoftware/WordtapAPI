package dashboard

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	dashQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/dashboard/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// DashboardRouter exposes executive governance dashboard endpoints.
type DashboardRouter struct {
	getAdminDashboard   *dashQuery.GetAdminDashboardHandler
	getTeacherDashboard *dashQuery.GetTeacherDashboardHandler
	listGlobalStudents  *dashQuery.ListGlobalStudentsHandler
	rbacEvaluator       *access.RBACPolicyEvaluator
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

// WithRBACEvaluator attaches the dynamic RBAC policy evaluator for directory scoping.
func (r *DashboardRouter) WithRBACEvaluator(eval *access.RBACPolicyEvaluator) *DashboardRouter {
	r.rbacEvaluator = eval
	return r
}

// RegisterRoutes registers dashboard routes on v1.
func (r *DashboardRouter) RegisterRoutes(v1 *echo.Group, authRequired, adminDashboardAccess echo.MiddlewareFunc) {
	v1.GET("/admin/dashboard", r.GetAdminDashboard, authRequired, adminDashboardAccess)
}

// RegisterTeacherRoutes registers teacher dashboard metrics route on v1.
func (r *DashboardRouter) RegisterTeacherRoutes(v1 *echo.Group, authRequired, instructorOrAdmin, globalStudentsAccess echo.MiddlewareFunc) {
	if r.getTeacherDashboard != nil {
		v1.GET("/teacher/dashboard", r.GetTeacherDashboard, authRequired, instructorOrAdmin)
	}
	if r.listGlobalStudents != nil {
		v1.GET("/teacher/global-students", r.ListGlobalStudents, authRequired, globalStudentsAccess)
	}
}

// ListGlobalStudents returns filtered global student telemetry from MySQL.
func (r *DashboardRouter) ListGlobalStudents(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	role := entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr)))

	eval := r.rbacEvaluator
	if eval == nil {
		eval = access.NewRBACPolicyEvaluator(nil)
	}

	var instructorID string
	if !eval.IsAllowed(c.Request().Context(), role, access.PermStudentsViewDirectory) {
		trimmedID := strings.TrimSpace(userID)
		if role != entities.RoleInstructor || trimmedID == "" {
			return response.Fail(c, http.StatusForbidden, "No tienes permisos para realizar esta acción.")
		}
		instructorID = trimmedID
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	data, err := r.listGlobalStudents.Handle(c.Request().Context(), dashQuery.GlobalStudentFilter{
		InstructorID: instructorID,
		Query:        c.QueryParam("q"),
		Plan:         c.QueryParam("plan"),
		Level:        c.QueryParam("level"),
		CourseID:     c.QueryParam("course_id"),
		Status:       c.QueryParam("status"),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el listado global de estudiantes.", err)
	}
	return c.JSON(http.StatusOK, data)
}

// GetTeacherDashboard returns real-time learning, vault, and revenue KPIs from MySQL.
func (r *DashboardRouter) GetTeacherDashboard(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	role := entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr)))

	var scopeInstructorID string
	switch role {
	case entities.RoleAdmin:
		scopeInstructorID = strings.TrimSpace(c.QueryParam("instructor_id"))
	case entities.RoleInstructor:
		scopeInstructorID = strings.TrimSpace(userID)
		if scopeInstructorID == "" {
			return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
		}
	default:
		return response.Fail(c, http.StatusForbidden, "No tienes permisos para realizar esta acción.")
	}

	period := c.QueryParam("period")
	data, err := r.getTeacherDashboard.Handle(c.Request().Context(), scopeInstructorID, period)
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
