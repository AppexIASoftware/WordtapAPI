package gamification

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// CreateAdminGameRequest is the payload of the platform-global activity
// creation endpoint. There is no course_id: global activities are not linked to
// a course.
type CreateAdminGameRequest struct {
	CategoryID       *string `json:"category_id"`
	BaseArchetype    string  `json:"base_archetype"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	Level            string  `json:"level"`
	AccessTier       *string `json:"access_tier"`
	TimeLimitSeconds int     `json:"time_limit_seconds"`
	AllowedMistakes  int     `json:"allowed_mistakes"`
	RewardXP         int     `json:"reward_xp"`
	ItemCount        int     `json:"item_count"`
	SourceType       string  `json:"source_type"`
	SourceID         *string `json:"source_id"`
	CustomWordsJSON  *string `json:"custom_words_json"`
}

// UpdateAdminGameRequest is the payload of the platform-global activity update
// endpoint. There is no course_id: global activities are not linked to a course.
type UpdateAdminGameRequest struct {
	CategoryID       *string `json:"category_id"`
	BaseArchetype    string  `json:"base_archetype"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	Level            string  `json:"level"`
	AccessTier       *string `json:"access_tier"`
	TimeLimitSeconds *int    `json:"time_limit_seconds"`
	AllowedMistakes  *int    `json:"allowed_mistakes"`
	RewardXP         *int    `json:"reward_xp"`
	ItemCount        *int    `json:"item_count"`
	SourceType       string  `json:"source_type"`
	SourceID         *string `json:"source_id"`
	CustomWordsJSON  *string `json:"custom_words_json"`
	IsActive         *bool   `json:"is_active"`
}

// CreateAdminGame handles POST /api/v1/admin/games and creates a platform-global
// activity. The caller is recorded only as author; the activity belongs to the
// platform.
func (r *GameRouter) CreateAdminGame(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req CreateAdminGameRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	var accessTier entities.AccessTier
	if req.AccessTier != nil {
		accessTier = entities.AccessTier(*req.AccessTier)
	}

	cmd := commands.CreateGlobalGameCommand{
		AuthorID:         userID,
		CategoryID:       req.CategoryID,
		BaseArchetype:    req.BaseArchetype,
		Title:            req.Title,
		Description:      req.Description,
		Level:            req.Level,
		AccessTier:       accessTier,
		TimeLimitSeconds: req.TimeLimitSeconds,
		AllowedMistakes:  req.AllowedMistakes,
		RewardXP:         req.RewardXP,
		ItemCount:        req.ItemCount,
		SourceType:       req.SourceType,
		SourceID:         req.SourceID,
		CustomWordsJSON:  req.CustomWordsJSON,
	}

	game, err := r.createGlobalGameHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo crear la actividad.", err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"success": true,
		"data":    formatTeacherGameDTO(game),
	})
}

// UpdateAdminGame handles PUT /api/v1/admin/games/:id and edits a
// platform-global activity.
func (r *GameRouter) UpdateAdminGame(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	id := c.Param("id")
	if id == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la actividad.")
	}

	var req UpdateAdminGameRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	var accessTier *entities.AccessTier
	if req.AccessTier != nil {
		tier := entities.AccessTier(*req.AccessTier)
		accessTier = &tier
	}

	cmd := commands.UpdateGlobalGameCommand{
		GameID:           id,
		AuthorID:         userID,
		CategoryID:       req.CategoryID,
		BaseArchetype:    req.BaseArchetype,
		Title:            req.Title,
		Description:      req.Description,
		Level:            req.Level,
		AccessTier:       accessTier,
		TimeLimitSeconds: req.TimeLimitSeconds,
		AllowedMistakes:  req.AllowedMistakes,
		RewardXP:         req.RewardXP,
		ItemCount:        req.ItemCount,
		SourceType:       req.SourceType,
		SourceID:         req.SourceID,
		CustomWordsJSON:  req.CustomWordsJSON,
		IsActive:         req.IsActive,
	}

	game, err := r.updateGlobalGameHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar la actividad.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    formatTeacherGameDTO(game),
	})
}

// DeleteAdminGame handles DELETE /api/v1/admin/games/:id. Activities linked to
// lessons are archived instead of deleted, and the response reports which
// happened together with the usage count.
func (r *GameRouter) DeleteAdminGame(c *echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la actividad.")
	}

	result, err := r.deleteGlobalGameHandler.Handle(c.Request().Context(), commands.DeleteGlobalGameCommand{
		GameID: id,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar la actividad.", err)
	}

	message := "Actividad eliminada correctamente."
	if result.Archived {
		message = "Actividad archivada correctamente porque está en uso en lecciones."
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"message": message,
		"data": map[string]any{
			"archived":    result.Archived,
			"usage_count": result.UsageCount,
		},
	})
}
