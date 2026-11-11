package device

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type DeviceRouter struct {
	repo domainRepo.DeviceRepository
}

func NewDeviceRouter(repo domainRepo.DeviceRepository) *DeviceRouter {
	return &DeviceRouter{repo: repo}
}

func (r *DeviceRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	devices := v1.Group("/devices")
	devices.Use(authRequired)
	devices.POST("/register", r.Register)
	devices.DELETE("/token", r.Revoke)
}

type RegisterDeviceRequest struct {
	PushToken  string  `json:"push_token"`
	Platform   string  `json:"platform"`
	DeviceName *string `json:"device_name"`
}

func (r *DeviceRouter) Register(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Autenticación requerida.")
	}

	var req RegisterDeviceRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Datos de dispositivo inválidos.", err)
	}

	token := strings.TrimSpace(req.PushToken)
	if token == "" {
		return response.Fail(c, http.StatusBadRequest, "El token push es obligatorio.")
	}
	if len(token) > 512 {
		return response.Fail(c, http.StatusBadRequest, "El token push es demasiado largo.")
	}

	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if platform == "" {
		platform = "android"
	}

	device := &entities.UserDevice{
		UserID:     userID,
		PushToken:  &token,
		Platform:   &platform,
		DeviceName: req.DeviceName,
	}

	if err := r.repo.RegisterDevice(c.Request().Context(), device); err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo registrar el dispositivo.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Dispositivo registrado exitosamente.",
	})
}

func (r *DeviceRouter) Revoke(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Autenticación requerida.")
	}

	token := strings.TrimSpace(c.QueryParam("token"))
	if token == "" {
		return response.Fail(c, http.StatusBadRequest, "El token push a revocar es obligatorio.")
	}

	if err := r.repo.RevokeToken(c.Request().Context(), userID, token); err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo revocar el token.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Token revocado exitosamente.",
	})
}
