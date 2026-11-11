package vault

import (
	"net/http"

	"github.com/labstack/echo/v5"

	vaultCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type DispatchRouter struct {
	handler *vaultCmd.DispatchVaultWordHandler
}

func NewDispatchRouter(handler *vaultCmd.DispatchVaultWordHandler) *DispatchRouter {
	return &DispatchRouter{handler: handler}
}

func (r *DispatchRouter) RegisterRoutes(v1 *echo.Group, authRequired, instructorOrAdmin echo.MiddlewareFunc) {
	group := v1.Group("/vault/items")
	group.Use(authRequired, instructorOrAdmin)
	group.POST("/:id/dispatch", r.Dispatch)
}

func (r *DispatchRouter) Dispatch(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Autenticación requerida.")
	}

	roleStr, _ := c.Get("role").(string)
	userRole := entities.UserRole(roleStr)

	itemID := c.Param("id")
	var body struct {
		TargetAudience    string  `json:"target_audience"`
		TargetClassroomID *string `json:"target_classroom_id"`
		TargetCEFRLevel   *string `json:"target_cefr_level"`
	}
	if err := c.Bind(&body); err != nil {
		return response.Fail(c, http.StatusBadRequest, "Datos de entrada inválidos.")
	}

	cmd := vaultCmd.DispatchVaultWordCommand{
		ItemVaultID:       itemID,
		TargetAudience:    body.TargetAudience,
		TargetClassroomID: body.TargetClassroomID,
		TargetCEFRLevel:   body.TargetCEFRLevel,
		SenderUserID:      userID,
		SenderRole:        userRole,
	}

	res, err := r.handler.Handle(c.Request().Context(), cmd)
	if err != nil {
		if err == vaultCmd.ErrVaultItemNotFound {
			return response.Fail(c, http.StatusNotFound, err.Error())
		}
		if err == vaultCmd.ErrClassroomForbidden {
			return response.Fail(c, http.StatusForbidden, err.Error())
		}
		return response.Fail(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, res)
}
