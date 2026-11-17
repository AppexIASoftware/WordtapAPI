package classroom

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

func (r *ClassroomRouter) GetClassroomStumbles(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	roleStr, _ := c.Get("user_role").(string)

	classroomID := c.Param("id")
	if classroomID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del aula.")
	}

	stumbles, err := r.getStumbleListHandler.Handle(c.Request().Context(), queries.GetStumbleListQuery{
		ClassroomID:   classroomID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr))),
	})
	if err != nil {
		if errors.Is(err, queries.ErrClassroomNotFound) {
			return response.Fail(c, http.StatusNotFound, err.Error())
		}
		if errors.Is(err, queries.ErrClassroomForbidden) {
			return response.Fail(c, http.StatusForbidden, err.Error())
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la lista de tropiezos.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    stumbles,
	})
}
