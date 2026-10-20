package gamification

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type GameRouter struct {
	createHandler           *commands.CreateTeacherGameHandler
	updateHandler           *commands.UpdateTeacherGameHandler
	deleteHandler           *commands.DeleteTeacherGameHandler
	listHandler             *queries.ListTeacherGamesHandler
	getHandler              *queries.GetTeacherGameDetailHandler
	listAdminGamesHandler   *queries.ListAdminGamesHandler
	createGlobalGameHandler *commands.CreateGlobalGameHandler
	updateGlobalGameHandler *commands.UpdateGlobalGameHandler
	deleteGlobalGameHandler *commands.DeleteGlobalGameHandler
}

func NewGameRouter(
	createHandler *commands.CreateTeacherGameHandler,
	updateHandler *commands.UpdateTeacherGameHandler,
	deleteHandler *commands.DeleteTeacherGameHandler,
	listHandler *queries.ListTeacherGamesHandler,
	getHandler *queries.GetTeacherGameDetailHandler,
	listAdminGamesHandler *queries.ListAdminGamesHandler,
	createGlobalGameHandler *commands.CreateGlobalGameHandler,
	updateGlobalGameHandler *commands.UpdateGlobalGameHandler,
	deleteGlobalGameHandler *commands.DeleteGlobalGameHandler,
) *GameRouter {
	return &GameRouter{
		createHandler:           createHandler,
		updateHandler:           updateHandler,
		deleteHandler:           deleteHandler,
		listHandler:             listHandler,
		getHandler:              getHandler,
		listAdminGamesHandler:   listAdminGamesHandler,
		createGlobalGameHandler: createGlobalGameHandler,
		updateGlobalGameHandler: updateGlobalGameHandler,
		deleteGlobalGameHandler: deleteGlobalGameHandler,
	}
}

func (r *GameRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
	adminOrModerator echo.MiddlewareFunc,
) {
	// Admin global games management (platform-wide activities only)
	v1.GET("/admin/games", r.ListAdminGames, authRequired, adminOrModerator)
	v1.POST("/admin/games", r.CreateAdminGame, authRequired, adminOrModerator)
	v1.PUT("/admin/games/:id", r.UpdateAdminGame, authRequired, adminOrModerator)
	v1.DELETE("/admin/games/:id", r.DeleteAdminGame, authRequired, adminOrModerator)

	group := v1.Group("/teacher/games")
	group.Use(authRequired, instructorOrAdmin)

	group.POST("", r.CreateGame)
	group.GET("", r.ListGames)
	group.GET("/:id", r.GetGameDetail)
	group.PUT("/:id", r.UpdateGame)
	group.DELETE("/:id", r.DeleteGame)
}

type CreateGameRequest struct {
	CourseID         *string `json:"course_id"`
	BaseArchetype    string  `json:"base_archetype"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	Level            string  `json:"level"`
	TimeLimitSeconds int     `json:"time_limit_seconds"`
	AllowedMistakes  int     `json:"allowed_mistakes"`
	RewardXP         int     `json:"reward_xp"`
	ItemCount        int     `json:"item_count"`
	SourceType       string  `json:"source_type"`
	SourceID         *string `json:"source_id"`
	CustomWordsJSON  *string `json:"custom_words_json"`
}

type UpdateGameRequest struct {
	CourseID         *string `json:"course_id"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	Level            string  `json:"level"`
	TimeLimitSeconds int     `json:"time_limit_seconds"`
	AllowedMistakes  int     `json:"allowed_mistakes"`
	RewardXP         int     `json:"reward_xp"`
	ItemCount        int     `json:"item_count"`
	SourceType       string  `json:"source_type"`
	SourceID         *string `json:"source_id"`
	CustomWordsJSON  *string `json:"custom_words_json"`
}

func (r *GameRouter) CreateGame(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req CreateGameRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	cmd := commands.CreateTeacherGameCommand{
		TeacherID:        userID,
		CourseID:         req.CourseID,
		BaseArchetype:    req.BaseArchetype,
		Title:            req.Title,
		Description:      req.Description,
		Level:            req.Level,
		TimeLimitSeconds: req.TimeLimitSeconds,
		AllowedMistakes:  req.AllowedMistakes,
		RewardXP:         req.RewardXP,
		ItemCount:        req.ItemCount,
		SourceType:       req.SourceType,
		SourceID:         req.SourceID,
		CustomWordsJSON:  req.CustomWordsJSON,
	}

	game, err := r.createHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo crear la actividad.", err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"success": true,
		"data":    formatTeacherGameDTO(game),
	})
}

func (r *GameRouter) ListGames(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	games, err := r.listHandler.Handle(c.Request().Context(), queries.ListTeacherGamesQuery{
		TeacherID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar las actividades.", err)
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

// ListAdminGames handles GET /api/v1/admin/games (all teacher games across all authors)
func (r *GameRouter) ListAdminGames(c *echo.Context) error {
	games, err := r.listAdminGamesHandler.Handle(c.Request().Context(), queries.ListAdminGamesQuery{})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar las actividades globales.", err)
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

func (r *GameRouter) GetGameDetail(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	id := c.Param("id")
	if id == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la actividad.")
	}

	game, err := r.getHandler.Handle(c.Request().Context(), queries.GetTeacherGameDetailQuery{
		GameID:    id,
		TeacherID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusNotFound, "No se encontró la actividad.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    formatTeacherGameDTO(game),
	})
}

func (r *GameRouter) UpdateGame(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	id := c.Param("id")
	if id == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la actividad.")
	}

	var req UpdateGameRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	cmd := commands.UpdateTeacherGameCommand{
		GameID:           id,
		TeacherID:        userID,
		CourseID:         req.CourseID,
		Title:            req.Title,
		Description:      req.Description,
		Level:            req.Level,
		TimeLimitSeconds: req.TimeLimitSeconds,
		AllowedMistakes:  req.AllowedMistakes,
		RewardXP:         req.RewardXP,
		ItemCount:        req.ItemCount,
		SourceType:       req.SourceType,
		SourceID:         req.SourceID,
		CustomWordsJSON:  req.CustomWordsJSON,
	}

	game, err := r.updateHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar la actividad.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    formatTeacherGameDTO(game),
	})
}

func (r *GameRouter) DeleteGame(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	id := c.Param("id")
	if id == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la actividad.")
	}

	err := r.deleteHandler.Handle(c.Request().Context(), commands.DeleteTeacherGameCommand{
		GameID:    id,
		TeacherID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar la actividad.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"message": "Actividad eliminada correctamente.",
	})
}

func formatTeacherGameDTO(g *entities.TeacherGame) map[string]any {
	courseTitle := ""
	if g.Course != nil {
		courseTitle = g.Course.Title
	}

	teacherName := ""
	teacherEmail := ""
	if g.Teacher != nil {
		teacherName = g.Teacher.Name
		teacherEmail = g.Teacher.Email
	}

	return map[string]any{
		"id":                 g.ID,
		"teacher_id":         g.TeacherID,
		"teacher_name":       teacherName,
		"teacher_email":      teacherEmail,
		"course_id":          g.CourseID,
		"course_title":       courseTitle,
		"base_archetype":     g.BaseArchetype,
		"title":              g.Title,
		"description":        g.Description,
		"level":              g.Level,
		"time_limit_seconds": g.TimeLimitSeconds,
		"allowed_mistakes":   g.AllowedMistakes,
		"reward_xp":          g.RewardXP,
		"item_count":         g.ItemCount,
		"source_type":        g.SourceType,
		"source_id":          g.SourceID,
		"custom_words_json":  g.CustomWordsJSON,
		"is_global":          g.IsGlobal,
		"is_active":          g.IsActive,
		"created_at":         g.CreatedAt.Format("02/01/2006"),
	}
}
