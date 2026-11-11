package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	app "github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationprefs"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
	"github.com/labstack/echo/v5"
)

type Router struct{ repo app.Repository }

func NewRouter(repo app.Repository) *Router { return &Router{repo: repo} }

func (r *Router) RegisterRoutes(v1 *echo.Group, auth echo.MiddlewareFunc) {
	v1.GET("/notification-preferences", r.Get, auth)
	v1.PUT("/notification-preferences", r.Put, auth)
}

func (r *Router) Get(c *echo.Context) error {
	id, ok := c.Get("user_id").(string)
	if !ok || id == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	result, err := r.repo.Get(c.Request().Context(), id)
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, "No se pudieron cargar las preferencias.")
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": result})
}

func (r *Router) Put(c *echo.Context) error {
	id, ok := c.Get("user_id").(string)
	if !ok || id == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	var input app.Update
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 16385))
	if err != nil || len(body) > 16384 {
		return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
	}
	if err := input.Validate(); err != nil {
		return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
	}
	result, err := r.repo.Put(c.Request().Context(), id, input)
	if errors.Is(err, app.ErrInvalid) {
		return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
	}
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, "No se pudieron guardar las preferencias.")
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": result})
}
