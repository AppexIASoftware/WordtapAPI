package course

import (
	"errors"
	"net/http"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
	"github.com/labstack/echo/v5"
)

type SimulatedPurchaseRouter struct {
	handler *commands.SimulatePurchaseHandler
}

func NewSimulatedPurchaseRouter(handler *commands.SimulatePurchaseHandler) *SimulatedPurchaseRouter {
	return &SimulatedPurchaseRouter{handler: handler}
}

func (r *SimulatedPurchaseRouter) RegisterRoutes(v1 *echo.Group, auth, student echo.MiddlewareFunc) {
	v1.POST("/student/courses/:id/simulated-purchase", r.Purchase, auth, student)
	v1.GET("/student/courses/:id/simulated-purchase", r.Find, auth, student)
}

func (r *SimulatedPurchaseRouter) Purchase(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	p, err := r.handler.Purchase(c.Request().Context(), userID, c.Param("id"), c.Request().Header.Get("Idempotency-Key"))
	if err != nil {
		return simulatedPurchaseError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "simulated": true, "data": p})
}

func (r *SimulatedPurchaseRouter) Find(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	p, err := r.handler.Find(c.Request().Context(), userID, c.Param("id"))
	if err != nil {
		return simulatedPurchaseError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "simulated": true, "data": p})
}

func simulatedPurchaseError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, commands.ErrSimulationDisabled):
		return response.Fail(c, http.StatusNotFound, "La simulación no está disponible.")
	case errors.Is(err, commands.ErrInvalidPurchase):
		return response.Fail(c, http.StatusBadRequest, "Solicitud de compra inválida.")
	case errors.Is(err, commands.ErrCourseNotPurchasable):
		return response.Fail(c, http.StatusNotFound, "El curso no está disponible para compra.")
	case errors.Is(err, commands.ErrIdempotencyConflict):
		return response.Fail(c, http.StatusConflict, "La clave de solicitud ya corresponde a otro curso.")
	default:
		return response.Fail(c, http.StatusInternalServerError, "No se pudo procesar la solicitud.")
	}
}
