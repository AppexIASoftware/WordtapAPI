package gamification

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type GameStudentRouter struct {
	listStudentGamesHandler *queries.ListStudentTeacherGamesHandler
	getStudentGameHandler   *queries.GetStudentTeacherGameDetailHandler
}

func NewGameStudentRouter(
	listStudentGamesHandler *queries.ListStudentTeacherGamesHandler,
	getStudentGameHandler *queries.GetStudentTeacherGameDetailHandler,
) *GameStudentRouter {
	return &GameStudentRouter{
		listStudentGamesHandler: listStudentGamesHandler,
		getStudentGameHandler:   getStudentGameHandler,
	}
}

func (r *GameStudentRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
) {
	studentGamesGroup := v1.Group("/games/student")
	studentGamesGroup.Use(authRequired)
	studentGamesGroup.GET("", r.ListStudentGames)
	studentGamesGroup.GET("/:id", r.GetStudentGameDetail)
}

func (r *GameStudentRouter) ListStudentGames(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	games, err := r.listStudentGamesHandler.Handle(c.Request().Context(), queries.ListStudentTeacherGamesQuery{
		StudentID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar las actividades de tus docentes.", err)
	}

	payload := make([]map[string]any, 0, len(games))
	for _, g := range games {
		payload = append(payload, formatTeacherGameDTO(&g))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    payload,
	})
}

func (r *GameStudentRouter) GetStudentGameDetail(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	id := c.Param("id")
	if id == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la actividad.")
	}

	game, err := r.getStudentGameHandler.Handle(c.Request().Context(), queries.GetStudentTeacherGameDetailQuery{
		StudentID: userID,
		GameID:    id,
	})
	if err != nil {
		return response.Error(c, http.StatusNotFound, "No se encontró la actividad o no tienes acceso.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    formatTeacherGameDTO(game),
	})
}
