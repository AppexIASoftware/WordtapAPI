package lesson

import (
	"net/http"

	lessoncommands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
	"github.com/labstack/echo/v5"
)

type UpdateLessonGameActivityRequest struct {
	BaseArchetype    string  `json:"base_archetype"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	TimeLimitSeconds int     `json:"time_limit_seconds"`
	AllowedMistakes  int     `json:"allowed_mistakes"`
	RewardXP         int     `json:"reward_xp"`
	ItemCount        int     `json:"item_count"`
}

func (r *LessonGameRouter) UpdateActivity(c *echo.Context) error {
	var req UpdateLessonGameActivityRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)

	updated, err := r.updateActivityHandler.Handle(c.Request().Context(), lessoncommands.UpdateLessonGameActivityCommand{
		LessonID:         c.Param("id"),
		GameID:           c.Param("game_id"),
		RequesterID:      userID,
		RequesterRole:    entities.UserRole(role),
		BaseArchetype:    req.BaseArchetype,
		Title:            req.Title,
		Description:      req.Description,
		TimeLimitSeconds: req.TimeLimitSeconds,
		AllowedMistakes:  req.AllowedMistakes,
		RewardXP:         req.RewardXP,
		ItemCount:        req.ItemCount,
	})
	if err != nil {
		return writeLessonGameError(c, err, "No se pudo actualizar la actividad.")
	}

	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": updated})
}
