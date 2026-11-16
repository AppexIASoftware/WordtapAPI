package user

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	userCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	userQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/queries"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/auditlog"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// AdminStudentsRouter exposes admin supervision endpoints for student accounts.
type AdminStudentsRouter struct {
	detailHandler *userQuery.GetStudentDetailHandler
	statusHandler *userCmd.UpdateStudentStatusHandler
	auditRepo     domainRepo.AuditRepository
}

// NewAdminStudentsRouter creates an AdminStudentsRouter.
func NewAdminStudentsRouter(
	detailHandler *userQuery.GetStudentDetailHandler,
	statusHandler *userCmd.UpdateStudentStatusHandler,
) *AdminStudentsRouter {
	return &AdminStudentsRouter{
		detailHandler: detailHandler,
		statusHandler: statusHandler,
	}
}

// WithAuditRepo attaches an optional AuditRepository for student status mutations.
func (r *AdminStudentsRouter) WithAuditRepo(repo domainRepo.AuditRepository) *AdminStudentsRouter {
	r.auditRepo = repo
	return r
}

// RegisterRoutes mounts admin student endpoints under /api/v1/admin/students.
func (r *AdminStudentsRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	canViewStudents echo.MiddlewareFunc,
	canManageStatus echo.MiddlewareFunc,
) {
	group := v1.Group("/admin/students")
	group.Use(authRequired)
	group.GET("/:id", r.GetDetail, canViewStudents)
	group.PATCH("/:id/status", r.UpdateStatus, canManageStatus)
}

type updateStudentStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// GetDetail handles GET /api/v1/admin/students/:id.
func (r *AdminStudentsRouter) GetDetail(c *echo.Context) error {
	detail, err := r.detailHandler.Handle(c.Request().Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domainRepo.ErrStudentNotFound) {
			return response.Error(c, http.StatusNotFound, "Estudiante no encontrado.", err)
		}
		if errors.Is(err, userQuery.ErrInvalidStudentID) {
			return response.Error(c, http.StatusBadRequest, "Identificador de estudiante inválido.", err)
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo consultar el detalle del estudiante.", err)
	}
	return c.JSON(http.StatusOK, detail)
}

// UpdateStatus handles PATCH /api/v1/admin/students/:id/status.
func (r *AdminStudentsRouter) UpdateStatus(c *echo.Context) error {
	var req updateStudentStatusRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	result, err := r.statusHandler.Handle(c.Request().Context(), userCmd.UpdateStudentStatusCommand{
		StudentID: c.Param("id"),
		IsActive:  req.IsActive,
	})
	if err != nil {
		if errors.Is(err, domainRepo.ErrStudentNotFound) {
			return response.Error(c, http.StatusNotFound, "Estudiante no encontrado.", err)
		}
		if errors.Is(err, userCmd.ErrStudentIDRequired) {
			return response.Error(c, http.StatusBadRequest, "Identificador de estudiante inválido.", err)
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo actualizar el estado del estudiante.", err)
	}

	action := "student.suspended"
	summary := "suspendido"
	if result.IsActive {
		action = "student.reactivated"
		summary = "reactivado"
	}
	auditlog.Record(c, r.auditRepo, action, "user", result.ID, map[string]any{
		"email":   result.Email,
		"summary": "Estudiante " + summary + " desde el directorio admin",
	})

	return c.JSON(http.StatusOK, result)
}
