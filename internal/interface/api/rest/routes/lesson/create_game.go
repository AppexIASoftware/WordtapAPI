package lesson

import (
	"net/http"

	gamecommands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	lessoncommands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
	"github.com/labstack/echo/v5"
)

func (r *LessonGameRouter) Create(c *echo.Context) error {
	var game gamecommands.CreateTeacherGameCommand
	if err := c.Bind(&game); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	created, err := r.createHandler.Handle(c.Request().Context(), lessoncommands.CreateLessonGameCommand{
		LessonID: c.Param("id"), RequesterID: userID, RequesterRole: entities.UserRole(role), Game: game,
	})
	if err != nil {
		return writeLessonGameError(c, err, "No se pudo crear la actividad.")
	}
	return c.JSON(http.StatusCreated, map[string]any{"success": true, "data": created})
}
