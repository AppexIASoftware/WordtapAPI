package dashboard

import (
	"net/http"

	"github.com/labstack/echo/v5"

	dashQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/dashboard/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// DashboardRouter exposes executive governance dashboard endpoints.
type DashboardRouter struct {
	getAdminDashboard *dashQuery.GetAdminDashboardHandler
}

// NewDashboardRouter creates a DashboardRouter.
func NewDashboardRouter(getAdminDashboard *dashQuery.GetAdminDashboardHandler) *DashboardRouter {
	return &DashboardRouter{getAdminDashboard: getAdminDashboard}
}

// RegisterRoutes registers dashboard routes on v1.
func (r *DashboardRouter) RegisterRoutes(v1 *echo.Group, authRequired, adminOrModerator echo.MiddlewareFunc) {
	v1.GET("/admin/dashboard", r.GetAdminDashboard, authRequired, adminOrModerator)
}

// GetAdminDashboard returns real-time platform metrics and recent activity from MySQL.
func (r *DashboardRouter) GetAdminDashboard(c *echo.Context) error {
	data, err := r.getAdminDashboard.Handle(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el dashboard administrativo.", err)
	}
	return c.JSON(http.StatusOK, data)
}
