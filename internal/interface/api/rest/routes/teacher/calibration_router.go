package teacher

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

const (
	defaultEaseFactor  = 2.5
	defaultQuality     = 4
	defaultRepetitions = 3
)

type ReviewCalibrationPayload struct {
	EaseFactor  float64 `json:"ease_factor"`
	Quality     int     `json:"quality"`
	Repetitions int     `json:"repetitions"`
}

// CalibrationRouter gestiona la persistencia del calibrador pedagógico de repaso del docente.
type CalibrationRouter struct {
	repo repositories.ReviewCalibrationRepository
}

func NewCalibrationRouter(repo repositories.ReviewCalibrationRepository) *CalibrationRouter {
	return &CalibrationRouter{repo: repo}
}

func (r *CalibrationRouter) RegisterRoutes(v1 *echo.Group, authRequired, instructorOrAdmin echo.MiddlewareFunc) {
	group := v1.Group("/teacher", authRequired, instructorOrAdmin)
	group.GET("/review-calibration", r.GetCalibration)
	group.PUT("/review-calibration", r.SaveCalibration)
}

func (r *CalibrationRouter) GetCalibration(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	item, err := r.repo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la calibración pedagógica.", err)
	}
	if item == nil {
		return c.JSON(http.StatusOK, ReviewCalibrationPayload{
			EaseFactor:  defaultEaseFactor,
			Quality:     defaultQuality,
			Repetitions: defaultRepetitions,
		})
	}

	return c.JSON(http.StatusOK, ReviewCalibrationPayload{
		EaseFactor:  item.EaseFactor,
		Quality:     item.Quality,
		Repetitions: item.Repetitions,
	})
}

func (r *CalibrationRouter) SaveCalibration(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req ReviewCalibrationPayload
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	if req.EaseFactor < 1.3 || req.EaseFactor > 3.0 {
		return response.Fail(c, http.StatusBadRequest, "El factor de facilidad debe estar entre 1.3 y 3.0.")
	}
	if req.Quality < 0 || req.Quality > 5 {
		return response.Fail(c, http.StatusBadRequest, "La calidad de respuesta debe estar entre 0 y 5.")
	}
	if req.Repetitions < 1 || req.Repetitions > 10 {
		return response.Fail(c, http.StatusBadRequest, "El número de repeticiones debe estar entre 1 y 10.")
	}

	entity := &entities.TeacherReviewCalibration{
		UserID:      userID,
		EaseFactor:  req.EaseFactor,
		Quality:     req.Quality,
		Repetitions: req.Repetitions,
	}
	if err := r.repo.Upsert(c.Request().Context(), entity); err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo guardar la calibración pedagógica.", err)
	}

	return c.JSON(http.StatusOK, req)
}
