package notification

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationdispatch"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type AnnouncementRouter struct {
	announcementHandler *notificationdispatch.BroadcastAnnouncementHandler
	deviceRepo          domainRepo.DeviceRepository
}

func NewAnnouncementRouter(
	announcementHandler *notificationdispatch.BroadcastAnnouncementHandler,
	deviceRepo domainRepo.DeviceRepository,
) *AnnouncementRouter {
	return &AnnouncementRouter{
		announcementHandler: announcementHandler,
		deviceRepo:          deviceRepo,
	}
}

func (r *AnnouncementRouter) RegisterRoutes(v1 *echo.Group, authRequired, adminOrStaff echo.MiddlewareFunc) {
	announcementsGroup := v1.Group("/admin/announcements")
	announcementsGroup.Use(authRequired, adminOrStaff)
	announcementsGroup.POST("", r.Broadcast)

	statsGroup := v1.Group("/admin/notifications")
	statsGroup.Use(authRequired, adminOrStaff)
	statsGroup.GET("/stats", r.GetStats)
}

func (r *AnnouncementRouter) Broadcast(c *echo.Context) error {
	var cmd notificationdispatch.BroadcastAnnouncementCommand
	if err := c.Bind(&cmd); err != nil {
		return response.Fail(c, http.StatusBadRequest, "Datos de entrada inválidos.")
	}

	if adminID, ok := c.Get("user_id").(string); ok && adminID != "" {
		cmd.AdminID = adminID
	}

	res, err := r.announcementHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return response.Fail(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, res)
}

func (r *AnnouncementRouter) GetStats(c *echo.Context) error {
	count, err := r.deviceRepo.CountActiveTokens(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Error al consultar estadísticas de dispositivos.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"active_devices_count": count,
	})
}
