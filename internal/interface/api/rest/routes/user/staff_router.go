package user

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	userCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/commands"
	userQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/user/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/auditlog"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type StaffRouter struct {
	listHandler      *userQuery.ListStaffHandler
	provisionHandler *userCmd.ProvisionStaffHandler
	statusHandler    *userCmd.UpdateStaffStatusHandler
	deleteHandler    *userCmd.DeleteStaffHandler
	auditRepo        domainRepo.AuditRepository
}

func NewStaffRouter(
	listHandler *userQuery.ListStaffHandler,
	provisionHandler *userCmd.ProvisionStaffHandler,
	statusHandler *userCmd.UpdateStaffStatusHandler,
	deleteHandler *userCmd.DeleteStaffHandler,
) *StaffRouter {
	return &StaffRouter{
		listHandler:      listHandler,
		provisionHandler: provisionHandler,
		statusHandler:    statusHandler,
		deleteHandler:    deleteHandler,
	}
}

// WithAuditRepo attaches an optional AuditRepository for staff mutations.
func (r *StaffRouter) WithAuditRepo(repo domainRepo.AuditRepository) *StaffRouter {
	r.auditRepo = repo
	return r
}

func (r *StaffRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	adminOrModerator echo.MiddlewareFunc,
	adminOnly echo.MiddlewareFunc,
) {
	staffGroup := v1.Group("/admin/staff")
	staffGroup.Use(authRequired)
	staffGroup.GET("", r.ListStaff, adminOrModerator)
	staffGroup.POST("", r.ProvisionStaff, adminOnly)
	staffGroup.PATCH("/:id/status", r.UpdateStatus, adminOnly)
	staffGroup.DELETE("/:id", r.DeleteStaff, adminOnly)
}

type ProvisionStaffRequest struct {
	Email string            `json:"email"`
	Name  string            `json:"name"`
	Role  entities.UserRole `json:"role"`
}

type UpdateStaffStatusRequest struct {
	IsActive bool `json:"is_active"`
}

func (r *StaffRouter) ListStaff(c *echo.Context) error {
	staff, err := r.listHandler.Handle(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo consultar el directorio de staff.", err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"staff": staff,
	})
}

func (r *StaffRouter) ProvisionStaff(c *echo.Context) error {
	requesterID, _ := c.Get("user_id").(string)
	var req ProvisionStaffRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	result, err := r.provisionHandler.Handle(c.Request().Context(), userCmd.ProvisionStaffCommand{
		RequesterID: requesterID,
		Email:       req.Email,
		Name:        req.Name,
		Role:        req.Role,
	})
	if err != nil {
		if errors.Is(err, userCmd.ErrInvalidStaffEmail) || errors.Is(err, userCmd.ErrInvalidStaffRole) {
			return response.Fail(c, http.StatusBadRequest, err.Error())
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo provisionar el usuario de staff.", err)
	}

	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	auditlog.Record(c, r.auditRepo, "STAFF_PROVISION", "staff", result.User.ID, map[string]any{
		"summary": "Provisionó o actualizó miembro del equipo",
		"email":   req.Email,
		"role":    req.Role,
		"created": result.Created,
	})
	return c.JSON(status, result)
}

func (r *StaffRouter) UpdateStatus(c *echo.Context) error {
	requesterID, _ := c.Get("user_id").(string)
	targetID := c.Param("id")

	var req UpdateStaffStatusRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	updated, err := r.statusHandler.Handle(c.Request().Context(), userCmd.UpdateStaffStatusCommand{
		RequesterID:  requesterID,
		TargetUserID: targetID,
		IsActive:     req.IsActive,
	})
	if err != nil {
		if errors.Is(err, userCmd.ErrCannotSuspendSelf) || errors.Is(err, userCmd.ErrTargetUserMissing) {
			return response.Fail(c, http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Fail(c, http.StatusNotFound, "Usuario no encontrado.")
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo actualizar el estado del usuario.", err)
	}

	auditlog.Record(c, r.auditRepo, "STAFF_STATUS_UPDATE", "staff", targetID, map[string]any{
		"summary":   "Actualizó estado de acceso del miembro",
		"is_active": req.IsActive,
	})
	return c.JSON(http.StatusOK, map[string]any{
		"user": updated,
	})
}

func (r *StaffRouter) DeleteStaff(c *echo.Context) error {
	requesterID, _ := c.Get("user_id").(string)
	targetID := c.Param("id")

	err := r.deleteHandler.Handle(c.Request().Context(), userCmd.DeleteStaffCommand{
		RequesterID:  requesterID,
		TargetUserID: targetID,
	})
	if err != nil {
		if errors.Is(err, userCmd.ErrCannotDeleteSelf) || errors.Is(err, userCmd.ErrTargetUserMissing) {
			return response.Fail(c, http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, userCmd.ErrUserHasBillingRecords) {
			return response.Fail(c, http.StatusConflict, err.Error())
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Fail(c, http.StatusNotFound, "Usuario no encontrado.")
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo eliminar el usuario.", err)
	}

	auditlog.Record(c, r.auditRepo, "STAFF_DELETE", "staff", targetID, map[string]any{
		"summary": "Eliminó miembro del directorio de staff",
	})
	return c.JSON(http.StatusOK, map[string]any{
		"deleted": true,
	})
}
