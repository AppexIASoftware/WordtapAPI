package gamification

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type ProgressRouter struct {
	finishHandler  *commands.RecordGameVictoryHandler
	summaryHandler *queries.GetUserProgressHandler
}

func NewProgressRouter(
	finishHandler *commands.RecordGameVictoryHandler,
	summaryHandler *queries.GetUserProgressHandler,
) *ProgressRouter {
	return &ProgressRouter{
		finishHandler:  finishHandler,
		summaryHandler: summaryHandler,
	}
}

func (r *ProgressRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	group := v1.Group("/gamification/progress")
	group.Use(authRequired)

	group.POST("/finish", r.FinishGameProgress)
	group.GET("/summary", r.GetProgressSummary)
}

type FinishGameProgressRequest struct {
	LessonID       string `json:"lesson_id"`
	StarsEarned    int    `json:"stars_earned"`
	WordsPracticed int    `json:"words_practiced"`
}

// FinishGameProgress maneja POST /api/v1/gamification/progress/finish
func (r *ProgressRouter) FinishGameProgress(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req FinishGameProgressRequest
	if err := c.Bind(&req); err != nil {
		return response.Fail(c, http.StatusBadRequest, "Datos de progreso inválidos.")
	}

	result, err := r.finishHandler.Handle(c.Request().Context(), commands.RecordGameVictoryCommand{
		UserID:         userID,
		LessonID:       req.LessonID,
		StarsEarned:    req.StarsEarned,
		WordsPracticed: req.WordsPracticed,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo registrar el progreso del juego.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    result,
	})
}

// GetProgressSummary maneja GET /api/v1/gamification/progress/summary
func (r *ProgressRouter) GetProgressSummary(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	summary, err := r.summaryHandler.Handle(c.Request().Context(), queries.GetUserProgressQuery{
		UserID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo obtener el resumen de progreso.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    summary,
	})
}
