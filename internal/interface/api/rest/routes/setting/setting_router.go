package setting

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	settingCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/commands"
	settingQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/auditlog"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type SettingRouter struct {
	getPublicHandler *settingQuery.GetPublicSettingsHandler
	getAllHandler    *settingQuery.GetAllSettingsHandler
	setHandler       *settingCmd.SetPlatformSettingHandler
	auditRepo        domainRepo.AuditRepository
	rbacEvaluator    *access.RBACPolicyEvaluator
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

// WithAuditRepo attaches an optional AuditRepository for platform setting updates.
func (r *SettingRouter) WithAuditRepo(repo domainRepo.AuditRepository) *SettingRouter {
	r.auditRepo = repo
	return r
}

// WithRBACEvaluator attaches the dynamic RBAC policy evaluator for /rbac/me lookups.
func (r *SettingRouter) WithRBACEvaluator(eval *access.RBACPolicyEvaluator) *SettingRouter {
	r.rbacEvaluator = eval
	return r
}

func (r *SettingRouter) RegisterRoutes(
	v1 *echo.Group,
	authMiddleware echo.MiddlewareFunc,
	adminOnly echo.MiddlewareFunc,
) {
	// Endpoint público
	v1.GET("/public-settings", r.GetPublicSettings)

	// Permisos efectivos del usuario autenticado
	v1.GET("/rbac/me", r.GetMyRBACPermissions, authMiddleware)

	// Endpoints administrativos
	adminGroup := v1.Group("/admin/settings")
	adminGroup.Use(authMiddleware, adminOnly)
	adminGroup.GET("", r.GetAllSettings)
	adminGroup.PUT("", r.UpdateSetting)
}

// GetMyRBACPermissions returns the evaluated RBAC permission map for the authenticated user's role.
func (r *SettingRouter) GetMyRBACPermissions(c *echo.Context) error {
	roleStr, _ := c.Get("user_role").(string)
	role := entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr)))
	var perms map[string]bool
	if r.rbacEvaluator != nil {
		perms = r.rbacEvaluator.PermissionsForRole(c.Request().Context(), role)
	} else {
		perms = map[string]bool{}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"role":        string(role),
		"permissions": perms,
	})
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

	auditlog.Record(c, r.auditRepo, "SETTING_UPDATE", "platform_setting", req.Key, map[string]any{
		"summary": "Actualizó configuración de plataforma / monetización",
		"key":     req.Key,
		"value":   req.Value,
	})
	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"key":    req.Key,
		"value":  req.Value,
	})
}
