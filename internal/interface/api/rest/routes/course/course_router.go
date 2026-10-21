package course

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type CourseRouter struct {
	createCourseHandler         *courseCmd.CreateCourseHandler
	updateCourseHandler         *courseCmd.UpdateCourseHandler
	deleteCourseHandler         *courseCmd.DeleteCourseHandler
	submitCourseReviewHandler   *courseCmd.SubmitCourseReviewHandler
	listTeacherCoursesHandler   *courseQuery.ListTeacherCoursesHandler
	listPublishedCoursesHandler *courseQuery.ListPublishedCoursesHandler
	listAdminCoursesHandler     *courseQuery.ListAdminCoursesHandler
	getCourseDetailHandler      *courseQuery.GetCourseDetailHandler
}

func NewCourseRouter(
	createCourseHandler *courseCmd.CreateCourseHandler,
	updateCourseHandler *courseCmd.UpdateCourseHandler,
	deleteCourseHandler *courseCmd.DeleteCourseHandler,
	submitCourseReviewHandler *courseCmd.SubmitCourseReviewHandler,
	listTeacherCoursesHandler *courseQuery.ListTeacherCoursesHandler,
	listPublishedCoursesHandler *courseQuery.ListPublishedCoursesHandler,
	listAdminCoursesHandler *courseQuery.ListAdminCoursesHandler,
	getCourseDetailHandler *courseQuery.GetCourseDetailHandler,
) *CourseRouter {
	return &CourseRouter{
		createCourseHandler:         createCourseHandler,
		updateCourseHandler:         updateCourseHandler,
		deleteCourseHandler:         deleteCourseHandler,
		submitCourseReviewHandler:   submitCourseReviewHandler,
		listTeacherCoursesHandler:   listTeacherCoursesHandler,
		listPublishedCoursesHandler: listPublishedCoursesHandler,
		listAdminCoursesHandler:     listAdminCoursesHandler,
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
	adminOrModerator echo.MiddlewareFunc,
) {
	// Course catalog (staff only: admin, moderator. Teacher has no access to Wordtap catalog)
	v1.GET("/courses", r.ListPublicCourses, authRequired, catalogRoles)

	// Course details (accessible by author, admin, moderator)
	v1.GET("/courses/:id", r.GetCourseDetail, authRequired, staffRoles)

	// Admin global courses list (all courses, any author, all statuses)
	v1.GET("/admin/courses", r.ListAdminCourses, authRequired, adminOrModerator)

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
	Status        *string `json:"status"`
	CoverImageURL *string `json:"cover_image_url"`
	ChangeSummary *string `json:"change_summary"`
}

// CreateCourse handles POST /api/v1/courses
func (r *CourseRouter) CreateCourse(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req CreateCourseRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	roleStr, _ := c.Get("user_role").(string)
	ownerType := entities.OwnerTypeTeacher
	if entities.UserRole(roleStr) == entities.RoleAdmin {
		ownerType = entities.OwnerTypePlatform
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
		OwnerType:     ownerType,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo crear el curso.", err)
	}

	return c.JSON(http.StatusCreated, course)
}

// UpdateCourse handles PUT /api/v1/courses/:id
func (r *CourseRouter) UpdateCourse(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	var req UpdateCourseRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	var tierPtr *entities.AccessTier
	if req.AccessTier != nil {
		t := entities.AccessTier(*req.AccessTier)
		tierPtr = &t
	}

	var statusPtr *entities.ContentStatus
	if req.Status != nil && *req.Status != "" {
		st := entities.ContentStatus(*req.Status)
		statusPtr = &st
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
		Status:        statusPtr,
		ChangeSummary: req.ChangeSummary,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar el curso.", err)
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteCourse handles DELETE /api/v1/courses/:id
func (r *CourseRouter) DeleteCourse(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
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
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar el curso.", err)
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
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
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
				return response.Fail(c, http.StatusNotFound, "No se encontró el curso.")
			}
			return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver este curso.")
		}
		return response.Fail(c, http.StatusNotFound, "No se encontró el curso.")
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
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
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
			return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver este curso.")
		}
		return response.Fail(c, http.StatusNotFound, "No se encontró el curso.")
	}

	isOwner := course.CreatedBy != nil && *course.CreatedBy == userID
	isAdmin := entities.UserRole(roleStr) == entities.RoleAdmin
	if !isOwner && !isAdmin {
		return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver este curso.")
	}

	return c.JSON(http.StatusOK, course)
}

// ListTeacherCourses handles GET /api/v1/teacher/courses
func (r *CourseRouter) ListTeacherCourses(c *echo.Context) error {
	userIDVal := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	ctx := c.Request().Context()
	courses, err := r.listTeacherCoursesHandler.Handle(ctx, courseQuery.ListTeacherCoursesQuery{
		AuthorID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar tus cursos.", err)
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
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
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
		return response.Error(c, http.StatusBadRequest, "No se pudo enviar el curso a revisión.", err)
	}

	return c.JSON(http.StatusOK, submitted)
}

// ListPublicCourses handles GET /api/v1/courses (public catalog of published courses)
func (r *CourseRouter) ListPublicCourses(c *echo.Context) error {
	ctx := c.Request().Context()
	courses, err := r.listPublishedCoursesHandler.Handle(ctx, courseQuery.ListPublishedCoursesQuery{})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el catálogo de cursos.", err)
	}

	publicCourses := make([]PublicCourseDTO, 0, len(courses))
	for _, c := range courses {
		publicCourses = append(publicCourses, ToPublicCourseDTO(&c))
	}

	return c.JSON(http.StatusOK, publicCourses)
}

// ListAdminCourses handles GET /api/v1/admin/courses (all courses across all authors and statuses)
func (r *CourseRouter) ListAdminCourses(c *echo.Context) error {
	ctx := c.Request().Context()
	courses, err := r.listAdminCoursesHandler.Handle(ctx, courseQuery.ListAdminCoursesQuery{})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la lista administrativa de cursos.", err)
	}

	return c.JSON(http.StatusOK, courses)
}
