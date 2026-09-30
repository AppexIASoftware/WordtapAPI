package lesson

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

// LessonRouter expone los endpoints REST para consultas y gestión de lecciones.
type LessonRouter struct {
	getLessonDetailHandler *queries.GetLessonDetailHandler
}

// NewLessonRouter inicializa el router de lecciones inyectando el repositorio correspondiente.
func NewLessonRouter(repo repositories.LessonRepository) *LessonRouter {
	return &LessonRouter{
		getLessonDetailHandler: queries.NewGetLessonDetailHandler(repo),
	}
}

// RegisterRoutes registra los endpoints del catálogo y lecciones.
func (r *LessonRouter) RegisterRoutes(v1 *echo.Group) {
	lessonsGroup := v1.Group("/lessons")
	lessonsGroup.GET("/:id", r.GetLessonDetail)
}

// GetLessonDetail maneja la solicitud GET /api/services/v1/lessons/:id
func (r *LessonRouter) GetLessonDetail(c *echo.Context) error {
	identifier := c.Param("id")
	if identifier == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "missing lesson identifier",
		})
	}

	ctx := c.Request().Context()
	dto, err := r.getLessonDetailHandler.Handle(ctx, queries.GetLessonDetailQuery{Identifier: identifier})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]any{
				"error": "lesson not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "failed to fetch lesson detail",
		})
	}

	return c.JSON(http.StatusOK, dto)
}
