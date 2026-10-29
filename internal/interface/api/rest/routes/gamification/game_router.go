package gamification

import (
	"net/http"
	"strings"

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
	listGlobalGamesHandler  *queries.ListGlobalGamesHandler
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
	listGlobalGamesHandler *queries.ListGlobalGamesHandler,
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
		listGlobalGamesHandler:  listGlobalGamesHandler,
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
	// Public global games catalog for authenticated users (students, teachers, admins)
	v1.GET("/games/global", r.ListGlobalGames, authRequired)

	// Admin global games management (platform-wide activities only)
	v1.GET("/admin/games", r.ListAdminGames, authRequired, adminOrModerator)
	v1.POST("/admin/games", r.CreateAdminGame, authRequired, adminOrModerator)
	v1.PUT("/admin/games/:id", r.UpdateAdminGame, authRequired, adminOrModerator)
	v1.DELETE("/admin/games/:id", r.DeleteAdminGame, authRequired, adminOrModerator)

	group := v1.Group("/teacher/games")
	group.Use(authRequired, instructorOrAdmin)

	group.POST("", r.CreateGame)
	group.GET("", r.ListGames)
	group.GET("/global", r.ListGlobalGames)
	group.GET("/:id", r.GetGameDetail)
	group.PUT("/:id", r.UpdateGame)
	group.DELETE("/:id", r.DeleteGame)
}

type CreateGameRequest struct {
	CourseID              *string  `json:"course_id"`
	CategoryID            *string  `json:"category_id"`
	BaseArchetype         string   `json:"base_archetype"`
	Title                 string   `json:"title"`
	Description           *string  `json:"description"`
	Level                 string   `json:"level"`
	AccessTier            *string  `json:"access_tier"`
	TimeLimitSeconds      int      `json:"time_limit_seconds"`
	AllowedMistakes       int      `json:"allowed_mistakes"`
	RewardXP              int      `json:"reward_xp"`
	ItemCount             int      `json:"item_count"`
	SourceType            string   `json:"source_type"`
	SourceID              *string  `json:"source_id"`
	CustomWordsJSON       *string  `json:"custom_words_json"`
	IsPublishedStandalone *bool    `json:"is_published_standalone"`
	AudienceScope         *string  `json:"audience_scope"`
	AssignedCourseIDs     []string `json:"assigned_course_ids"`
}

type UpdateGameRequest struct {
	CourseID              *string   `json:"course_id"`
	CategoryID            *string   `json:"category_id"`
	BaseArchetype         string    `json:"base_archetype"`
	Title                 string    `json:"title"`
	Description           *string   `json:"description"`
	Level                 string    `json:"level"`
	AccessTier            *string   `json:"access_tier"`
	TimeLimitSeconds      int       `json:"time_limit_seconds"`
	AllowedMistakes       int       `json:"allowed_mistakes"`
	RewardXP              int       `json:"reward_xp"`
	ItemCount             int       `json:"item_count"`
	SourceType            string    `json:"source_type"`
	SourceID              *string   `json:"source_id"`
	CustomWordsJSON       *string   `json:"custom_words_json"`
	IsPublishedStandalone *bool     `json:"is_published_standalone"`
	AudienceScope         *string   `json:"audience_scope"`
	AssignedCourseIDs     *[]string `json:"assigned_course_ids"`
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
	if strings.TrimSpace(req.SourceType) == "course_lesson" && req.SourceID != nil && strings.TrimSpace(*req.SourceID) != "" {
		return response.Fail(c, http.StatusBadRequest, "Crea la actividad desde la lección correspondiente.")
	}

	var accessTier entities.AccessTier
	if req.AccessTier != nil {
		accessTier = entities.AccessTier(*req.AccessTier)
	}

	isPublishedStandalone := false
	if req.IsPublishedStandalone != nil {
		isPublishedStandalone = *req.IsPublishedStandalone
	}
	audienceScope := "all_students"
	if req.AudienceScope != nil && strings.TrimSpace(*req.AudienceScope) != "" {
		audienceScope = strings.TrimSpace(*req.AudienceScope)
	}

	cmd := commands.CreateTeacherGameCommand{
		TeacherID:             userID,
		CourseID:              req.CourseID,
		CategoryID:            req.CategoryID,
		BaseArchetype:         req.BaseArchetype,
		Title:                 req.Title,
		Description:           req.Description,
		Level:                 req.Level,
		AccessTier:            accessTier,
		TimeLimitSeconds:      req.TimeLimitSeconds,
		AllowedMistakes:        req.AllowedMistakes,
		RewardXP:              req.RewardXP,
		ItemCount:             req.ItemCount,
		SourceType:            req.SourceType,
		SourceID:              req.SourceID,
		CustomWordsJSON:       req.CustomWordsJSON,
		IsPublishedStandalone: isPublishedStandalone,
		AudienceScope:         audienceScope,
		AssignedCourseIDs:     req.AssignedCourseIDs,
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

// ListGlobalGames handles GET /api/v1/teacher/games/global. Every instructor
// can read the platform catalog, but only administrators can modify it.
func (r *GameRouter) ListGlobalGames(c *echo.Context) error {
	games, err := r.listGlobalGamesHandler.Handle(c.Request().Context(), queries.ListGlobalGamesQuery{})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar los juegos globales.", err)
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

	var accessTier *entities.AccessTier
	if req.AccessTier != nil {
		tier := entities.AccessTier(*req.AccessTier)
		accessTier = &tier
	}

	cmd := commands.UpdateTeacherGameCommand{
		GameID:           id,
		TeacherID:        userID,
		CourseID:         req.CourseID,
		CategoryID:       req.CategoryID,
		BaseArchetype:    req.BaseArchetype,
		Title:            req.Title,
		Description:      req.Description,
		Level:            req.Level,
		AccessTier:       accessTier,
		TimeLimitSeconds: req.TimeLimitSeconds,
		AllowedMistakes:  req.AllowedMistakes,
		RewardXP:         req.RewardXP,
		ItemCount:             req.ItemCount,
		SourceType:            req.SourceType,
		SourceID:              req.SourceID,
		CustomWordsJSON:       req.CustomWordsJSON,
		IsPublishedStandalone: req.IsPublishedStandalone,
		AudienceScope:         req.AudienceScope,
		AssignedCourseIDs:     req.AssignedCourseIDs,
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
		"message": "Actividad eliminada o archivada correctamente.",
	})
}
