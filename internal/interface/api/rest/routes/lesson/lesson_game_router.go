package lesson

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	lessonCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	lessonQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// LessonGameRouter exposes lesson game linking endpoints.
type LessonGameRouter struct {
	listHandler   *lessonQuery.ListLessonGamesHandler
	linkHandler   *lessonCmd.LinkLessonGameHandler
	updateHandler *lessonCmd.UpdateLessonGameHandler
	unlinkHandler *lessonCmd.UnlinkLessonGameHandler
}

func NewLessonGameRouter(
	listHandler *lessonQuery.ListLessonGamesHandler,
	linkHandler *lessonCmd.LinkLessonGameHandler,
	updateHandler *lessonCmd.UpdateLessonGameHandler,
	unlinkHandler *lessonCmd.UnlinkLessonGameHandler,
) *LessonGameRouter {
	return &LessonGameRouter{
		listHandler:   listHandler,
		linkHandler:   linkHandler,
		updateHandler: updateHandler,
		unlinkHandler: unlinkHandler,
	}
}

func (r *LessonGameRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	staffRoles echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
) {
	v1.GET("/lessons/:id/games", r.List, authRequired)
	v1.POST("/lessons/:id/games", r.Link, authRequired, instructorOrAdmin)
	v1.PATCH("/lessons/:id/games/:game_id", r.Update, authRequired, instructorOrAdmin)
	v1.DELETE("/lessons/:id/games/:game_id", r.Unlink, authRequired, instructorOrAdmin)
}

type LinkLessonGameRequest struct {
	GameID     string `json:"game_id"`
	SortOrder  *int   `json:"sort_order"`
	IsRequired *bool  `json:"is_required"`
}

type UpdateLessonGameRequest struct {
	SortOrder  *int  `json:"sort_order"`
	IsRequired *bool `json:"is_required"`
}

// List handles GET /api/v1/lessons/:id/games.
func (r *LessonGameRouter) List(c *echo.Context) error {
	lessonID := c.Param("id")
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	dtos, err := r.listHandler.Handle(c.Request().Context(), lessonQuery.ListLessonGamesQuery{
		LessonID:      lessonID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return writeLessonGameError(c, err, "No se pudieron cargar los juegos de la lección.")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    dtos,
	})
}

// Link handles POST /api/v1/lessons/:id/games.
func (r *LessonGameRouter) Link(c *echo.Context) error {
	lessonID := c.Param("id")
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	var req LinkLessonGameRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	link, err := r.linkHandler.Handle(c.Request().Context(), lessonCmd.LinkLessonGameCommand{
		LessonID:      lessonID,
		GameID:        req.GameID,
		SortOrder:     req.SortOrder,
		IsRequired:    req.IsRequired,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return writeLessonGameError(c, err, "No se pudo vincular la actividad.")
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"success": true,
		"data":    lessonQuery.ToLessonGameDTO(*link),
	})
}

// Update handles PATCH /api/v1/lessons/:id/games/:game_id.
func (r *LessonGameRouter) Update(c *echo.Context) error {
	lessonID := c.Param("id")
	gameID := c.Param("game_id")
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	var req UpdateLessonGameRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	link, err := r.updateHandler.Handle(c.Request().Context(), lessonCmd.UpdateLessonGameCommand{
		LessonID:      lessonID,
		GameID:        gameID,
		SortOrder:     req.SortOrder,
		IsRequired:    req.IsRequired,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return writeLessonGameError(c, err, "No se pudo actualizar el vínculo.")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    lessonQuery.ToLessonGameDTO(*link),
	})
}

// Unlink handles DELETE /api/v1/lessons/:id/games/:game_id.
func (r *LessonGameRouter) Unlink(c *echo.Context) error {
	lessonID := c.Param("id")
	gameID := c.Param("game_id")
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	err := r.unlinkHandler.Handle(c.Request().Context(), lessonCmd.UnlinkLessonGameCommand{
		LessonID:      lessonID,
		GameID:        gameID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return writeLessonGameError(c, err, "No se pudo desvincular la actividad.")
	}

	return c.NoContent(http.StatusNoContent)
}

func writeLessonGameError(c *echo.Context, err error, fallback string) error {
	switch {
	case errors.Is(err, lessonCmd.ErrLessonGameLessonNotFound),
		errors.Is(err, lessonQuery.ErrLessonGamesNotFound):
		return response.Fail(c, http.StatusNotFound, "No se encontró la lección.")
	case errors.Is(err, lessonCmd.ErrLessonGameGameNotFound):
		return response.Fail(c, http.StatusNotFound, "No se encontró la actividad.")
	case errors.Is(err, lessonCmd.ErrLessonGameLinkNotFound):
		return response.Fail(c, http.StatusNotFound, "No se encontró el vínculo.")
	case errors.Is(err, lessonCmd.ErrLessonGameAccessDenied):
		return response.Fail(c, http.StatusForbidden, "No tienes permisos para modificar los juegos de esta lección.")
	case errors.Is(err, lessonQuery.ErrLessonGamesAccessDenied):
		return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver los juegos de esta lección.")
	case errors.Is(err, lessonCmd.ErrLessonGameDuplicate):
		return response.Fail(c, http.StatusConflict, "La actividad ya está vinculada a esta lección.")
	case errors.Is(err, lessonCmd.ErrLessonGameLevelTooHigh):
		return response.Fail(c, http.StatusUnprocessableEntity, "El nivel de la actividad supera el nivel del curso. Elige una actividad acorde o ajusta el nivel del curso.")
	default:
		return response.Error(c, http.StatusBadRequest, fallback, err)
	}
}
