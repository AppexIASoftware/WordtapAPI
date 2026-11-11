package notification

import (
	"net/http"
	"strconv"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type InboxRouter struct{ repo inbox.Repository }

func NewInboxRouter(repo inbox.Repository) *InboxRouter { return &InboxRouter{repo: repo} }
func (r *InboxRouter) RegisterRoutes(v1 *echo.Group, auth echo.MiddlewareFunc) {
	v1.GET("/notifications", r.List, auth)
	v1.GET("/notifications/unread-count", r.UnreadCount, auth)
	v1.PUT("/notifications/:id/read", r.MarkRead, auth)
}

func (r *InboxRouter) List(c *echo.Context) error {
	user, ok := c.Get("user_id").(string)
	if !ok || user == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	limit := 20
	if text := c.QueryParam("limit"); text != "" {
		value, err := strconv.Atoi(text)
		if err != nil || value < 1 || value > 100 {
			return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
		}
		limit = value
	}
	cursor, err := inbox.DecodeCursor(c.QueryParam("cursor"))
	if err != nil {
		return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
	}
	page, err := r.repo.List(c.Request().Context(), user, limit, cursor)
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, "No se pudieron cargar las notificaciones.")
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": page})
}

func (r *InboxRouter) UnreadCount(c *echo.Context) error {
	user, ok := c.Get("user_id").(string)
	if !ok || user == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	count, err := r.repo.UnreadCount(c.Request().Context(), user)
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, "No se pudieron cargar las notificaciones.")
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": map[string]any{"unread_count": count}})
}

func (r *InboxRouter) MarkRead(c *echo.Context) error {
	user, ok := c.Get("user_id").(string)
	if !ok || user == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil || len(id) != 36 {
		return response.Fail(c, http.StatusBadRequest, "Los datos enviados no son válidos.")
	}
	found, err := r.repo.MarkRead(c.Request().Context(), user, id)
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, "No se pudo actualizar la notificación.")
	}
	if !found {
		return response.Fail(c, http.StatusNotFound, "No se encontró la notificación.")
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true})
}
