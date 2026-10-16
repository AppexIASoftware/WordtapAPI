package course

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type CourseReviewRouter struct {
	submit   *commands.SubmitCourseReviewHandler
	withdraw *commands.WithdrawCourseReviewHandler
	review   *commands.ReviewCourseHandler
	pending  *queries.ListCourseReviewsHandler
	history  *queries.ListTeacherCourseReviewsHandler
}

func NewCourseReviewRouter(submit *commands.SubmitCourseReviewHandler, withdraw *commands.WithdrawCourseReviewHandler, review *commands.ReviewCourseHandler, pending *queries.ListCourseReviewsHandler, history *queries.ListTeacherCourseReviewsHandler) *CourseReviewRouter {
	return &CourseReviewRouter{submit: submit, withdraw: withdraw, review: review, pending: pending, history: history}
}

func (r *CourseReviewRouter) RegisterRoutes(v1 *echo.Group, auth echo.MiddlewareFunc, teacher echo.MiddlewareFunc, admin echo.MiddlewareFunc) {
	v1.POST("/courses/:id/submit-review", r.Submit, auth, teacher)
	v1.POST("/courses/:id/withdraw-review", r.Withdraw, auth, teacher)
	v1.POST("/teacher/course-reviews/:id/withdraw", r.Withdraw, auth, teacher)
	v1.GET("/teacher/course-reviews", r.History, auth, teacher)
	v1.GET("/admin/course-reviews", r.Pending, auth, admin)
	v1.POST("/admin/course-reviews/:id/decision", r.Decide, auth, admin)
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
	return c.JSON(http.StatusOK, result)
}
