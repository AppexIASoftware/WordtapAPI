package course

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type CourseReviewRouter struct {
	submit       *commands.SubmitCourseReviewHandler
	withdraw     *commands.WithdrawCourseReviewHandler
	review       *commands.ReviewCourseHandler
	pending      *queries.ListCourseReviewsHandler
	history      *queries.ListTeacherCourseReviewsHandler
	adminHistory *queries.ListReviewHistoryHandler
	auditRepo    domainRepo.AuditRepository
}

func NewCourseReviewRouter(submit *commands.SubmitCourseReviewHandler, withdraw *commands.WithdrawCourseReviewHandler, review *commands.ReviewCourseHandler, pending *queries.ListCourseReviewsHandler, history *queries.ListTeacherCourseReviewsHandler, adminHistory *queries.ListReviewHistoryHandler) *CourseReviewRouter {
	return &CourseReviewRouter{submit: submit, withdraw: withdraw, review: review, pending: pending, history: history, adminHistory: adminHistory}
}

// WithAuditRepo attaches an optional AuditRepository to persist review decisions in audit_logs.
func (r *CourseReviewRouter) WithAuditRepo(repo domainRepo.AuditRepository) *CourseReviewRouter {
	r.auditRepo = repo
	return r
}

func (r *CourseReviewRouter) recordAudit(ctx context.Context, actorID, ip, action, entityID string, payload map[string]any) {
	if r.auditRepo == nil {
		return
	}
	raw, _ := json.Marshal(payload)
	var actorPtr, idPtr, ipPtr *string
	if strings.TrimSpace(actorID) != "" {
		actorPtr = &actorID
	}
	if strings.TrimSpace(entityID) != "" {
		idPtr = &entityID
	}
	if strings.TrimSpace(ip) != "" {
		ipPtr = &ip
	}
	_ = r.auditRepo.CreateAuditLog(ctx, &entities.AuditLog{
		ActorUserID: actorPtr,
		Action:      action,
		EntityType:  "course",
		EntityID:    idPtr,
		Changes:     string(raw),
		IPAddress:   ipPtr,
	})
}

func (r *CourseReviewRouter) RegisterRoutes(v1 *echo.Group, auth echo.MiddlewareFunc, teacher echo.MiddlewareFunc, admin echo.MiddlewareFunc) {
	v1.POST("/courses/:id/submit-review", r.Submit, auth, teacher)
	v1.POST("/courses/:id/withdraw-review", r.Withdraw, auth, teacher)
	v1.POST("/teacher/course-reviews/:id/withdraw", r.Withdraw, auth, teacher)
	v1.GET("/teacher/course-reviews", r.History, auth, teacher)
	v1.GET("/admin/course-reviews", r.Pending, auth, admin)
	v1.GET("/admin/course-reviews/history", r.AdminHistory, auth, admin)
	v1.POST("/admin/course-reviews/:id/decision", r.Decide, auth, admin)
	v1.POST("/admin/courses/:id/suspend-review", r.SuspendToReview, auth, admin)
}

func (r *CourseReviewRouter) SuspendToReview(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	var body SubmitCourseReviewRequestBody
	_ = c.Bind(&body)
	result, err := r.submit.Handle(c.Request().Context(), commands.SubmitCourseReviewCommand{
		CourseID:      c.Param("id"),
		RequesterID:   userID,
		RequesterRole: entities.UserRole(role),
		ChangeSummary: body.ChangeSummary,
	})
	if err != nil {
		return response.Fail(c, http.StatusBadRequest, err.Error())
	}
	r.recordAudit(c.Request().Context(), userID, c.RealIP(), "COURSE_SUSPEND_REVIEW", c.Param("id"), map[string]any{
		"summary":        "Suspendió publicación del curso y lo devolvió a revisión",
		"change_summary": body.ChangeSummary,
	})
	return c.JSON(http.StatusOK, result)
}

func (r *CourseReviewRouter) AdminHistory(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	if userID == "" || (entities.UserRole(role) != entities.RoleAdmin && entities.UserRole(role) != entities.RoleModerator) {
		return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver el historial de revisiones.")
	}
	items, err := r.adminHistory.Handle(c.Request().Context(), queries.ListCourseReviewsQuery{RequesterID: userID, RequesterRole: entities.UserRole(role)})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el historial de revisiones.", err)
	}
	return c.JSON(http.StatusOK, items)
}

type SubmitCourseReviewRequestBody struct {
	ChangeSummary *string `json:"change_summary"`
}

func (r *CourseReviewRouter) Submit(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	var body SubmitCourseReviewRequestBody
	_ = c.Bind(&body)
	result, err := r.submit.Handle(c.Request().Context(), commands.SubmitCourseReviewCommand{
		CourseID:      c.Param("id"),
		RequesterID:   userID,
		RequesterRole: entities.UserRole(role),
		ChangeSummary: body.ChangeSummary,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo enviar el curso a revisión.", err)
	}
	return c.JSON(http.StatusCreated, result)
}

func (r *CourseReviewRouter) Withdraw(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	result, err := r.withdraw.Handle(c.Request().Context(), commands.WithdrawCourseReviewCommand{
		ID:            c.Param("id"),
		RequesterID:   userID,
		RequesterRole: entities.UserRole(role),
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo retirar el curso de revisión.", err)
	}
	return c.JSON(http.StatusOK, result)
}

func (r *CourseReviewRouter) Pending(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	items, err := r.pending.Handle(c.Request().Context(), queries.ListCourseReviewsQuery{RequesterID: userID, RequesterRole: entities.UserRole(role)})
	if err != nil {
		return response.Error(c, http.StatusForbidden, "No tienes permisos para ver las revisiones pendientes.", err)
	}
	return c.JSON(http.StatusOK, items)
}

func (r *CourseReviewRouter) History(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	items, err := r.history.Handle(c.Request().Context(), queries.ListTeacherCourseReviewsQuery{InstructorID: userID, RequesterID: userID, RequesterRole: entities.UserRole(role)})
	if err != nil {
		return response.Error(c, http.StatusForbidden, "No tienes permisos para ver tu historial de revisiones.", err)
	}
	return c.JSON(http.StatusOK, items)
}

type CourseReviewDecisionRequest struct {
	Status        entities.CourseReviewStatus `json:"status"`
	FeedbackNotes *string                     `json:"feedback_notes"`
}

func (r *CourseReviewRouter) Decide(c *echo.Context) error {
	var body CourseReviewDecisionRequest
	if err := c.Bind(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	result, err := r.review.Handle(c.Request().Context(), commands.ReviewCourseCommand{ReviewID: c.Param("id"), ReviewerID: userID, ReviewerRole: entities.UserRole(role), Status: body.Status, FeedbackNotes: body.FeedbackNotes})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo registrar la decisión de revisión.", err)
	}
	r.recordAudit(c.Request().Context(), userID, c.RealIP(), "COURSE_"+strings.ToUpper(string(body.Status)), c.Param("id"), map[string]any{
		"summary":        "Emitió dictamen de revisión curricular",
		"status":         body.Status,
		"feedback_notes": body.FeedbackNotes,
	})
	return c.JSON(http.StatusOK, result)
}
