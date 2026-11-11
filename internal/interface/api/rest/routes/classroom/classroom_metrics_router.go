package classroom

import (
	"net/http"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
	"github.com/labstack/echo/v5"
)

func (r *ClassroomRouter) GetClassroomStumbles(c *echo.Context) error {
	classroomID := c.Param("id")
	if classroomID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del aula.")
	}

	stumbles, err := r.getStumbleListHandler.Handle(c.Request().Context(), queries.GetStumbleListQuery{
		ClassroomID: classroomID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la lista de tropiezos.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    stumbles,
	})
}
