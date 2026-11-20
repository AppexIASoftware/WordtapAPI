package teacher

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type UpdateTeacherProfileRequest struct {
	Bio       *string `json:"bio"`
	Specialty *string `json:"specialty"`
}

// ProfileRouter gestiona el perfil académico y credenciales del docente.
type ProfileRouter struct {
	userRepo  repositories.UserRepository
	onUpdated func(userID string)
}

func NewProfileRouter(userRepo repositories.UserRepository, onUpdated func(userID string)) *ProfileRouter {
	return &ProfileRouter{
		userRepo:  userRepo,
		onUpdated: onUpdated,
	}
}

func (r *ProfileRouter) RegisterRoutes(v1 *echo.Group, authRequired, instructorOrAdmin echo.MiddlewareFunc) {
	group := v1.Group("/teacher", authRequired, instructorOrAdmin)
	group.PUT("/profile", r.UpdateProfile)
}

// UpdateProfile maneja PUT /api/v1/teacher/profile.
func (r *ProfileRouter) UpdateProfile(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req UpdateTeacherProfileRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	handler := commands.NewUpdateTeacherProfileHandler(r.userRepo)
	updated, err := handler.Handle(c.Request().Context(), commands.UpdateTeacherProfileCommand{
		UserID:    userID,
		Bio:       req.Bio,
		Specialty: req.Specialty,
	})
	if err != nil {
		if errors.Is(err, commands.ErrInvalidTeacherBio) || errors.Is(err, commands.ErrInvalidTeacherSpecialty) {
			return response.Fail(c, http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Fail(c, http.StatusNotFound, "No se encontró el usuario.")
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo actualizar tu perfil académico.", err)
	}

	if r.onUpdated != nil {
		r.onUpdated(userID)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user": updated,
	})
}
