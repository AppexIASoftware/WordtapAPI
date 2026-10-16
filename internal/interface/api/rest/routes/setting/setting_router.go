package setting

import (
	"net/http"

	"github.com/labstack/echo/v5"

	settingCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/commands"
	settingQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type SettingRouter struct {
	getPublicHandler *settingQuery.GetPublicSettingsHandler
	getAllHandler    *settingQuery.GetAllSettingsHandler
	setHandler       *settingCmd.SetPlatformSettingHandler
}

func NewSettingRouter(
	getPublicHandler *settingQuery.GetPublicSettingsHandler,
	getAllHandler *settingQuery.GetAllSettingsHandler,
	setHandler *settingCmd.SetPlatformSettingHandler,
) *SettingRouter {
	return &SettingRouter{
		getPublicHandler: getPublicHandler,
		getAllHandler:    getAllHandler,
		setHandler:       setHandler,
	}
}

func (r *SettingRouter) RegisterRoutes(
	v1 *echo.Group,
	authMiddleware echo.MiddlewareFunc,
	adminOrModerator echo.MiddlewareFunc,
) {
	// Endpoint público
	v1.GET("/public-settings", r.GetPublicSettings)

	// Endpoints administrativos
	adminGroup := v1.Group("/admin/settings")
	adminGroup.Use(authMiddleware, adminOrModerator)
	adminGroup.GET("", r.GetAllSettings)
	adminGroup.PUT("", r.UpdateSetting)
}

func (r *SettingRouter) GetPublicSettings(c *echo.Context) error {
	ctx := c.Request().Context()
	settings, err := r.getPublicHandler.Handle(ctx)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la configuración pública.", err)
	}
	return c.JSON(http.StatusOK, settings)
}

func (r *SettingRouter) GetAllSettings(c *echo.Context) error {
	ctx := c.Request().Context()
	settings, err := r.getAllHandler.Handle(ctx)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la configuración de la plataforma.", err)
	}
	return c.JSON(http.StatusOK, settings)
}

type UpdateSettingRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (r *SettingRouter) UpdateSetting(c *echo.Context) error {
	var req UpdateSettingRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	ctx := c.Request().Context()
	if err := r.setHandler.Handle(ctx, settingCmd.SetPlatformSettingCommand{
		Key:   req.Key,
		Value: req.Value,
	}); err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar la configuración.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"key":    req.Key,
		"value":  req.Value,
	})
}
