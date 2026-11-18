package course

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	courseCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type CourseRouter struct {
	createCourseHandler         *courseCmd.CreateCourseHandler
	updateCourseHandler         *courseCmd.UpdateCourseHandler
	deleteCourseHandler         *courseCmd.DeleteCourseHandler
	listTeacherCoursesHandler   *courseQuery.ListTeacherCoursesHandler
	listPublishedCoursesHandler *courseQuery.ListPublishedCoursesHandler
	listAdminCoursesHandler     *courseQuery.ListAdminCoursesHandler
	getCourseDetailHandler      *courseQuery.GetCourseDetailHandler
	contentPolicy               *access.ContentPolicy
}

func (r *CourseRouter) WithContentPolicy(policy *access.ContentPolicy) *CourseRouter {
	r.contentPolicy = policy
	return r
}

func NewCourseRouter(
	createCourseHandler *courseCmd.CreateCourseHandler,
	updateCourseHandler *courseCmd.UpdateCourseHandler,
	deleteCourseHandler *courseCmd.DeleteCourseHandler,
	listTeacherCoursesHandler *courseQuery.ListTeacherCoursesHandler,
	listPublishedCoursesHandler *courseQuery.ListPublishedCoursesHandler,
	listAdminCoursesHandler *courseQuery.ListAdminCoursesHandler,
	getCourseDetailHandler *courseQuery.GetCourseDetailHandler,
) *CourseRouter {
	return &CourseRouter{
		createCourseHandler:         createCourseHandler,
		updateCourseHandler:         updateCourseHandler,
		deleteCourseHandler:         deleteCourseHandler,
		listTeacherCoursesHandler:   listTeacherCoursesHandler,
		listPublishedCoursesHandler: listPublishedCoursesHandler,
		listAdminCoursesHandler:     listAdminCoursesHandler,
		getCourseDetailHandler:      getCourseDetailHandler,
	}
}

func requesterFromContext(c *echo.Context) (string, entities.UserRole) {
	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)
	return strings.TrimSpace(userID), entities.UserRole(strings.ToLower(strings.TrimSpace(roleStr)))
}

// RegisterRoutes registers course endpoints on the v1 router group.
func (r *CourseRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	catalogRoles echo.MiddlewareFunc,
	studentOrStaff echo.MiddlewareFunc,
	staffRoles echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
	adminOrModerator echo.MiddlewareFunc,
) {
	v1.GET("/courses", r.ListPublicCourses, authRequired, catalogRoles)
	v1.GET("/courses/:id", r.GetCourseDetail, authRequired, studentOrStaff)
	v1.GET("/admin/courses", r.ListAdminCourses, authRequired, adminOrModerator)

	coursesGroup := v1.Group("/courses")
	coursesGroup.Use(authRequired, staffRoles)
	coursesGroup.POST("", r.CreateCourse)
	coursesGroup.PUT("/:id", r.UpdateCourse)
	coursesGroup.DELETE("/:id", r.DeleteCourse)

	teacherGroup := v1.Group("/teacher")
	teacherGroup.Use(authRequired, instructorOrAdmin)
	teacherGroup.GET("/courses", r.ListTeacherCourses)
	teacherGroup.GET("/courses/:id", r.GetTeacherCourseDetail)
}

type CreateCourseRequest struct {
	Title         string  `json:"title"`
	Description   *string `json:"description"`
	Level         string  `json:"level"`
	CategoryID    *string `json:"category_id"`
	AccessTier    string  `json:"access_tier"`
	PriceCents    *int    `json:"price_cents"`
	SourceLang    string  `json:"source_lang"`
	TargetLang    string  `json:"target_lang"`
	CoverImageURL *string `json:"cover_image_url"`
}

type UpdateCourseRequest struct {
	Title         *string                 `json:"title"`
	Description   *string                 `json:"description"`
	Level         *string                 `json:"level"`
	CategoryID    entities.OptionalString `json:"category_id"`
	AccessTier    *string                 `json:"access_tier"`
	PriceCents    *int                    `json:"price_cents"`
	Status        *string                 `json:"status"`
	CoverImageURL *string                 `json:"cover_image_url"`
	ChangeSummary *string                 `json:"change_summary"`
}

// CreateCourse handles POST /api/v1/courses.
func (r *CourseRouter) CreateCourse(c *echo.Context) error {
	userID, role := requesterFromContext(c)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req CreateCourseRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	ownerType := entities.OwnerTypeTeacher
	if role == entities.RoleAdmin || role == entities.RoleModerator {
		ownerType = entities.OwnerTypePlatform
	}

	priceCentsVal := 0
	if req.PriceCents != nil {
		priceCentsVal = *req.PriceCents
	}
	course, err := r.createCourseHandler.Handle(c.Request().Context(), courseCmd.CreateCourseCommand{
		Title:         req.Title,
		Description:   req.Description,
		Level:         req.Level,
		CategoryID:    req.CategoryID,
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

// UpdateCourse handles PUT /api/v1/courses/:id.
func (r *CourseRouter) UpdateCourse(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	userID, role := requesterFromContext(c)
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

	updated, err := r.updateCourseHandler.Handle(c.Request().Context(), courseCmd.UpdateCourseCommand{
		CourseID:      courseID,
		Title:         req.Title,
		Description:   req.Description,
		Level:         req.Level,
		CategoryID:    req.CategoryID.Value,
		CategoryIDSet: req.CategoryID.Set,
		CoverImageURL: req.CoverImageURL,
		AccessTier:    tierPtr,
		PriceCents:    req.PriceCents,
		Status:        statusPtr,
		ChangeSummary: req.ChangeSummary,
		RequesterID:   userID,
		RequesterRole: role,
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar el curso.", err)
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteCourse handles DELETE /api/v1/courses/:id.
func (r *CourseRouter) DeleteCourse(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	userID, role := requesterFromContext(c)
	if err := r.deleteCourseHandler.Handle(c.Request().Context(), courseCmd.DeleteCourseCommand{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: role,
	}); err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar el curso.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status":    "deleted",
		"course_id": courseID,
	})
}
