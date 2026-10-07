package course

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type CourseRouter struct {
	createCourseHandler         *courseCmd.CreateCourseHandler
	updateCourseHandler         *courseCmd.UpdateCourseHandler
	deleteCourseHandler         *courseCmd.DeleteCourseHandler
	submitCourseReviewHandler   *courseCmd.SubmitCourseReviewHandler
	listTeacherCoursesHandler   *courseQuery.ListTeacherCoursesHandler
	listPublishedCoursesHandler *courseQuery.ListPublishedCoursesHandler
	getCourseDetailHandler      *courseQuery.GetCourseDetailHandler
}

func NewCourseRouter(
	createCourseHandler *courseCmd.CreateCourseHandler,
	updateCourseHandler *courseCmd.UpdateCourseHandler,
	deleteCourseHandler *courseCmd.DeleteCourseHandler,
	submitCourseReviewHandler *courseCmd.SubmitCourseReviewHandler,
	listTeacherCoursesHandler *courseQuery.ListTeacherCoursesHandler,
	listPublishedCoursesHandler *courseQuery.ListPublishedCoursesHandler,
	getCourseDetailHandler *courseQuery.GetCourseDetailHandler,
) *CourseRouter {
	return &CourseRouter{
		createCourseHandler:         createCourseHandler,
		updateCourseHandler:         updateCourseHandler,
		deleteCourseHandler:         deleteCourseHandler,
		submitCourseReviewHandler:   submitCourseReviewHandler,
		listTeacherCoursesHandler:   listTeacherCoursesHandler,
		listPublishedCoursesHandler: listPublishedCoursesHandler,
		getCourseDetailHandler:      getCourseDetailHandler,
	}
}

// RegisterRoutes registers course endpoints on the v1 router group.
func (r *CourseRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	catalogRoles echo.MiddlewareFunc,
	staffRoles echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
) {
	// Course catalog (staff only: admin, moderator. Teacher has no access to Wordtap catalog)
	v1.GET("/courses", r.ListPublicCourses, authRequired, catalogRoles)

	// Course details (accessible by author, admin, moderator)
	v1.GET("/courses/:id", r.GetCourseDetail, authRequired, staffRoles)

	// Course management (teacher, admin, moderator)
	coursesGroup := v1.Group("/courses")
	coursesGroup.Use(authRequired, staffRoles)
	coursesGroup.POST("", r.CreateCourse)
	coursesGroup.PUT("/:id", r.UpdateCourse)
	coursesGroup.DELETE("/:id", r.DeleteCourse)

	// Teacher private courses list and detail (only own courses)
	teacherGroup := v1.Group("/teacher")
	teacherGroup.Use(authRequired, instructorOrAdmin)
	teacherGroup.GET("/courses", r.ListTeacherCourses)
	teacherGroup.GET("/courses/:id", r.GetTeacherCourseDetail)
}

type CreateCourseRequest struct {
	Title         string  `json:"title"`
	Description   *string `json:"description"`
	Level         string  `json:"level"`
	AccessTier    string  `json:"access_tier"`
	PriceCents    *int    `json:"price_cents"`
	SourceLang    string  `json:"source_lang"`
	TargetLang    string  `json:"target_lang"`
	CoverImageURL *string `json:"cover_image_url"`
}

type UpdateCourseRequest struct {
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	Level         *string `json:"level"`
	AccessTier    *string `json:"access_tier"`
	PriceCents    *int    `json:"price_cents"`
	CoverImageURL *string `json:"cover_image_url"`
	ChangeSummary *string `json:"change_summary"`
}

// CreateCourse handles POST /api/v1/courses
func (r *CourseRouter) CreateCourse(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error": "unauthorized context",
		})
	}

	var req CreateCourseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "invalid request body",
		})
	}

	ctx := c.Request().Context()
	priceCentsVal := 0
	if req.PriceCents != nil {
		priceCentsVal = *req.PriceCents
	}
	course, err := r.createCourseHandler.Handle(ctx, courseCmd.CreateCourseCommand{
		Title:         req.Title,
		Description:   req.Description,
		Level:         req.Level,
		AccessTier:    entities.AccessTier(req.AccessTier),
		PriceCents:    priceCentsVal,
		SourceLang:    req.SourceLang,
		TargetLang:    req.TargetLang,
		CoverImageURL: req.CoverImageURL,
		AuthorID:      userID,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to create course",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, course)
}

// UpdateCourse handles PUT /api/v1/courses/:id
func (r *CourseRouter) UpdateCourse(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "missing course id parameter",
		})
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	var req UpdateCourseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "invalid request body",
		})
	}

	var tierPtr *entities.AccessTier
	if req.AccessTier != nil {
		t := entities.AccessTier(*req.AccessTier)
		tierPtr = &t
	}

	ctx := c.Request().Context()
	updated, err := r.updateCourseHandler.Handle(ctx, courseCmd.UpdateCourseCommand{
		CourseID:      courseID,
		Title:         req.Title,
		Description:   req.Description,
		Level:         req.Level,
		CoverImageURL: req.CoverImageURL,
		AccessTier:    tierPtr,
		PriceCents:    req.PriceCents,
		ChangeSummary: req.ChangeSummary,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to update course",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteCourse handles DELETE /api/v1/courses/:id
func (r *CourseRouter) DeleteCourse(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "missing course id parameter",
		})
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	err := r.deleteCourseHandler.Handle(ctx, courseCmd.DeleteCourseCommand{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to delete course",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status":    "deleted",
		"course_id": courseID,
	})
}

// GetCourseDetail handles GET /api/v1/courses/:id
func (r *CourseRouter) GetCourseDetail(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "missing course id parameter",
		})
	}

	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	ctx := c.Request().Context()
	course, err := r.getCourseDetailHandler.Handle(ctx, courseQuery.GetCourseDetailQuery{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		if errors.Is(err, courseQuery.ErrCourseAccessDenied) {
			if userID == "" {
				return c.JSON(http.StatusNotFound, map[string]any{
					"error": "course not found",
				})
			}
			return c.JSON(http.StatusForbidden, map[string]any{
				"error": "access denied: course is not published",
			})
		}
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "course not found",
		})
	}

	isOwner := userID != "" && course.CreatedBy != nil && *course.CreatedBy == userID
	isAdmin := entities.UserRole(roleStr) == entities.RoleAdmin

	if !isOwner && !isAdmin {
		return c.JSON(http.StatusOK, ToPublicCourseDTO(course))
	}

	return c.JSON(http.StatusOK, course)
}

// GetTeacherCourseDetail handles GET /api/v1/teacher/courses/:id (private teacher-owned detail path)
func (r *CourseRouter) GetTeacherCourseDetail(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "missing course id parameter",
		})
	}

	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error": "unauthorized context",
		})
	}

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	course, err := r.getCourseDetailHandler.Handle(ctx, courseQuery.GetCourseDetailQuery{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		if errors.Is(err, courseQuery.ErrCourseAccessDenied) {
			return c.JSON(http.StatusForbidden, map[string]any{
				"error": "access denied: not the owner of this course",
			})
		}
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "course not found",
		})
	}

	isOwner := course.CreatedBy != nil && *course.CreatedBy == userID
	isAdmin := entities.UserRole(roleStr) == entities.RoleAdmin
	if !isOwner && !isAdmin {
		return c.JSON(http.StatusForbidden, map[string]any{
			"error": "access denied: not the owner of this course",
		})
	}

	return c.JSON(http.StatusOK, course)
}

// ListTeacherCourses handles GET /api/v1/teacher/courses
func (r *CourseRouter) ListTeacherCourses(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error": "unauthorized context",
		})
	}

	ctx := c.Request().Context()
	courses, err := r.listTeacherCoursesHandler.Handle(ctx, courseQuery.ListTeacherCoursesQuery{
		AuthorID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "failed to retrieve teacher courses",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, courses)
}

type SubmitCourseReviewRequest struct {
	ChangeSummary *string `json:"change_summary"`
}

// SubmitCourseReview handles POST /api/v1/courses/:id/submit-review
func (r *CourseRouter) SubmitCourseReview(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "missing course id parameter",
		})
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	var req SubmitCourseReviewRequest
	_ = c.Bind(&req)

	ctx := c.Request().Context()
	submitted, err := r.submitCourseReviewHandler.Handle(ctx, courseCmd.SubmitCourseReviewCommand{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
		ChangeSummary: req.ChangeSummary,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "failed to submit course for review",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, submitted)
}

// ListPublicCourses handles GET /api/v1/courses (public catalog of published courses)
func (r *CourseRouter) ListPublicCourses(c *echo.Context) error {
	ctx := c.Request().Context()
	courses, err := r.listPublishedCoursesHandler.Handle(ctx, courseQuery.ListPublishedCoursesQuery{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "failed to retrieve public course catalog",
			"details": err.Error(),
		})
	}

	publicCourses := make([]PublicCourseDTO, 0, len(courses))
	for _, c := range courses {
		publicCourses = append(publicCourses, ToPublicCourseDTO(&c))
	}

	return c.JSON(http.StatusOK, publicCourses)
}
