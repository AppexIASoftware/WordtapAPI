package gamification

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type LivesRouter struct {
	statsRepo repositories.UserLearningStatsRepository
}

func NewLivesRouter(statsRepo repositories.UserLearningStatsRepository) *LivesRouter {
	return &LivesRouter{
		statsRepo: statsRepo,
	}
}

func (r *LivesRouter) RegisterRoutes(v1 *echo.Group, authRequired echo.MiddlewareFunc) {
	group := v1.Group("/gamification/lives")
	group.Use(authRequired)

	group.GET("", r.GetLives)
	group.POST("/deduct", r.DeductLife)
	group.POST("/refill", r.RefillLives)
}

// GetLives maneja GET /api/v1/gamification/lives
func (r *LivesRouter) GetLives(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	stats, err := r.statsRepo.GetLearningStats(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron obtener las vidas del usuario.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"lives_remaining": stats.LivesRemaining,
			"max_lives":       stats.MaxLives,
			"total_stars":     stats.TotalStars,
			"total_points":    stats.TotalPoints,
		},
	})
}

// DeductLife maneja POST /api/v1/gamification/lives/deduct
func (r *LivesRouter) DeductLife(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	stats, err := r.statsRepo.DeductUserLife(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo actualizar la vida.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"lives_remaining": stats.LivesRemaining,
			"max_lives":       stats.MaxLives,
		},
	})
}

// RefillLives maneja POST /api/v1/gamification/lives/refill
func (r *LivesRouter) RefillLives(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	stats, err := r.statsRepo.RefillUserLives(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron recargar las vidas.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"lives_remaining": stats.LivesRemaining,
			"max_lives":       stats.MaxLives,
		},
	})
}
