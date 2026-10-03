package teacher_application

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	appCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/teacher_application/commands"
	appQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/teacher_application/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type TeacherApplicationRouter struct {
	applyHandler    *appCmd.ApplyTeacherHandler
	reviewHandler   *appCmd.ReviewApplicationHandler
	deleteHandler   *appCmd.DeleteApplicationHandler
	myStatusHandler *appQuery.GetMyApplicationHandler
	listHandler     *appQuery.ListApplicationsHandler
}

func NewTeacherApplicationRouter(
	applyHandler *appCmd.ApplyTeacherHandler,
	reviewHandler *appCmd.ReviewApplicationHandler,
	deleteHandler *appCmd.DeleteApplicationHandler,
	myStatusHandler *appQuery.GetMyApplicationHandler,
	listHandler *appQuery.ListApplicationsHandler,
) *TeacherApplicationRouter {
	return &TeacherApplicationRouter{
		applyHandler:    applyHandler,
		reviewHandler:   reviewHandler,
		deleteHandler:   deleteHandler,
		myStatusHandler: myStatusHandler,
		listHandler:     listHandler,
	}
}

func (r *TeacherApplicationRouter) RegisterRoutes(
	v1 *echo.Group,
	authMiddleware echo.MiddlewareFunc,
	adminOrModerator echo.MiddlewareFunc,
) {
	// Rutas para postulantes autenticados
	appGroup := v1.Group("/teacher-applications")
	appGroup.Use(authMiddleware)
	appGroup.POST("", r.Apply)
	appGroup.GET("/my-status", r.GetMyStatus)

	// Rutas de administración y moderación
	adminGroup := v1.Group("/admin/teacher-applications")
	adminGroup.Use(authMiddleware, adminOrModerator)
	adminGroup.GET("", r.List)
	adminGroup.POST("/:id/approve", r.Approve)
	adminGroup.POST("/:id/reject", r.Reject)
	adminGroup.POST("/:id/suspend", r.Suspend)
	adminGroup.POST("/:id/reactivate", r.Reactivate)
	adminGroup.DELETE("/:id", r.Delete)
}

type ApplyRequest struct {
	Bio       *string `json:"bio"`
	Specialty *string `json:"specialty"`
}

func (r *TeacherApplicationRouter) Apply(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{"error": "unauthorized context"})
	}

	var req ApplyRequest
	_ = c.Bind(&req)

	ctx := c.Request().Context()
	app, err := r.applyHandler.Handle(ctx, appCmd.ApplyTeacherCommand{
		UserID:    userID,
		Bio:       req.Bio,
		Specialty: req.Specialty,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to submit teacher application",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, app)
}

func (r *TeacherApplicationRouter) GetMyStatus(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{"error": "unauthorized context"})
	}

	ctx := c.Request().Context()
	status, err := r.myStatusHandler.Handle(ctx, appQuery.GetMyApplicationQuery{
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "failed to check application status",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, status)
}

func (r *TeacherApplicationRouter) List(c *echo.Context) error {
	statusParam := strings.TrimSpace(c.QueryParam("status"))
	var statusPtr *entities.ApplicationStatus
	if statusParam != "" {
		st := entities.ApplicationStatus(statusParam)
		statusPtr = &st
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	ctx := c.Request().Context()
	res, err := r.listHandler.Handle(ctx, appQuery.ListApplicationsQuery{
		Status: statusPtr,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "failed to list applications",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, res)
}

func (r *TeacherApplicationRouter) Approve(c *echo.Context) error {
	appID := c.Param("id")
	if appID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "application id is required"})
	}

	reviewerID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	ctx := c.Request().Context()
	app, err := r.reviewHandler.Handle(ctx, appCmd.ReviewApplicationCommand{
		ApplicationID: appID,
		ReviewerID:    reviewerID,
		ReviewerRole:  entities.UserRole(roleStr),
		Action:        "approve",
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to approve application",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, app)
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

func (r *TeacherApplicationRouter) Reject(c *echo.Context) error {
	appID := c.Param("id")
	if appID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "application id is required"})
	}

	var req RejectRequest
	_ = c.Bind(&req)

	reviewerID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	ctx := c.Request().Context()
	app, err := r.reviewHandler.Handle(ctx, appCmd.ReviewApplicationCommand{
		ApplicationID:   appID,
		ReviewerID:      reviewerID,
		ReviewerRole:    entities.UserRole(roleStr),
		Action:          "reject",
		RejectionReason: req.Reason,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to reject application",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, app)
}

type SuspendRequest struct {
	Reason string `json:"reason"`
}

func (r *TeacherApplicationRouter) Suspend(c *echo.Context) error {
	appID := c.Param("id")
	if appID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "application id is required"})
	}

	var req SuspendRequest
	_ = c.Bind(&req)

	reviewerID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	ctx := c.Request().Context()
	app, err := r.reviewHandler.Handle(ctx, appCmd.ReviewApplicationCommand{
		ApplicationID:   appID,
		ReviewerID:      reviewerID,
		ReviewerRole:    entities.UserRole(roleStr),
		Action:          "suspend",
		RejectionReason: req.Reason,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to suspend application",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, app)
}

func (r *TeacherApplicationRouter) Reactivate(c *echo.Context) error {
	appID := c.Param("id")
	if appID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "application id is required"})
	}

	reviewerID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	ctx := c.Request().Context()
	app, err := r.reviewHandler.Handle(ctx, appCmd.ReviewApplicationCommand{
		ApplicationID: appID,
		ReviewerID:    reviewerID,
		ReviewerRole:  entities.UserRole(roleStr),
		Action:        "reactivate",
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to reactivate application",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, app)
}

func (r *TeacherApplicationRouter) Delete(c *echo.Context) error {
	appID := c.Param("id")
	if appID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "application id is required"})
	}

	reviewerID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	ctx := c.Request().Context()
	if err := r.deleteHandler.Handle(ctx, appCmd.DeleteApplicationCommand{
		ApplicationID: appID,
		ReviewerID:    reviewerID,
		ReviewerRole:  entities.UserRole(roleStr),
	}); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to delete application",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{"message": "application deleted successfully"})
}
