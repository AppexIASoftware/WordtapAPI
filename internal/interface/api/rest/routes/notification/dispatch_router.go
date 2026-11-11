package notification

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationdispatch"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type DispatchRouter struct {
	dispatchHandler *notificationdispatch.DispatchDailyWordHandler
}

func NewDispatchRouter(dispatchHandler *notificationdispatch.DispatchDailyWordHandler) *DispatchRouter {
	return &DispatchRouter{
		dispatchHandler: dispatchHandler,
	}
}

func (r *DispatchRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	group := v1.Group("/notifications")
	group.Use(authRequired)
	group.POST("/test-dispatch", r.TestDispatch)
}

func (r *DispatchRouter) TestDispatch(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Autenticación requerida.")
	}

	result, err := r.dispatchHandler.Handle(c.Request().Context(), notificationdispatch.DispatchDailyWordCommand{
		UserID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo despachar el recordatorio de prueba.", err)
	}

	return c.JSON(http.StatusOK, result)
}
