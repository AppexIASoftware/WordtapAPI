package catalog

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/catalog/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// UnlockRouter expone el estado de desbloqueo progresivo del catálogo para el
// usuario autenticado.
type UnlockRouter struct {
	unlockStateHandler *queries.GetCatalogUnlockStateHandler
}

func NewUnlockRouter(unlockStateHandler *queries.GetCatalogUnlockStateHandler) *UnlockRouter {
	return &UnlockRouter{unlockStateHandler: unlockStateHandler}
}

func (r *UnlockRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	group := v1.Group("/catalog")
	group.Use(authRequired)

	group.GET("/unlock-state", r.GetUnlockState)
}

// GetUnlockState maneja GET /api/v1/catalog/unlock-state
func (r *UnlockRouter) GetUnlockState(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	state, err := r.unlockStateHandler.Handle(c.Request().Context(), queries.GetCatalogUnlockStateQuery{
		UserID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo obtener el estado del catálogo.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    state,
	})
}
