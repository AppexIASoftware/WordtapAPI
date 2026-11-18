package course

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	courseQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

// GetCourseDetail handles GET /api/v1/courses/:id.
func (r *CourseRouter) GetCourseDetail(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	userID, role := requesterFromContext(c)
	ctx := c.Request().Context()
	course, err := r.getCourseDetailHandler.Handle(ctx, courseQuery.GetCourseDetailQuery{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: role,
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

	isOwner := access.EffectiveOwnerType(course) == entities.OwnerTypeTeacher && userID != "" && course.CreatedBy != nil && *course.CreatedBy == userID
	isAdmin := role == entities.RoleAdmin
	if role == entities.RoleInstructor && !isOwner {
		return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver este curso.")
	}
	canAudit := isOwner || isAdmin || (role == entities.RoleModerator &&
		(course.Status == entities.ContentStatusInReview || access.EffectiveOwnerType(course) == entities.OwnerTypePlatform))
	auditMode := c.QueryParam("audit") == "1" || c.QueryParam("audit") == "true"

	if (course.Status != entities.ContentStatusPublished || auditMode) && canAudit {
		return c.JSON(http.StatusOK, toAuditCourseDTO(course))
	}

	if !isOwner && !isAdmin {
		err := r.contentPolicy.Authorize(ctx, course, nil, userID, role)
		if err != nil && !errors.Is(err, access.ErrContentDenied) {
			return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el curso.", err)
		}
		return c.JSON(http.StatusOK, toPublicCourseDTO(course, err == nil))
	}

	return c.JSON(http.StatusOK, toPublicCourseDTO(course, true))
}

// GetTeacherCourseDetail handles GET /api/v1/teacher/courses/:id (private teacher-owned detail path).
func (r *CourseRouter) GetTeacherCourseDetail(c *echo.Context) error {
	courseID := c.Param("id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	userID, role := requesterFromContext(c)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	course, err := r.getCourseDetailHandler.Handle(c.Request().Context(), courseQuery.GetCourseDetailQuery{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: role,
	})
	if err != nil {
		if errors.Is(err, courseQuery.ErrCourseAccessDenied) {
			return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver este curso.")
		}
		return response.Fail(c, http.StatusNotFound, "No se encontró el curso.")
	}

	isOwner := access.EffectiveOwnerType(course) == entities.OwnerTypeTeacher && course.CreatedBy != nil && *course.CreatedBy == userID
	if !isOwner && role != entities.RoleAdmin {
		return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver este curso.")
	}

	return c.JSON(http.StatusOK, course)
}

// ListTeacherCourses handles GET /api/v1/teacher/courses.
func (r *CourseRouter) ListTeacherCourses(c *echo.Context) error {
	userID, _ := requesterFromContext(c)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	courses, err := r.listTeacherCoursesHandler.Handle(c.Request().Context(), courseQuery.ListTeacherCoursesQuery{
		AuthorID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar tus cursos.", err)
	}

	return c.JSON(http.StatusOK, courses)
}

// ListPublicCourses handles GET /api/v1/courses (public catalog of published courses).
func (r *CourseRouter) ListPublicCourses(c *echo.Context) error {
	courses, err := r.listPublishedCoursesHandler.Handle(c.Request().Context(), courseQuery.ListPublishedCoursesQuery{})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar el catálogo de cursos.", err)
	}

	publicCourses := make([]PublicCourseDTO, 0, len(courses))
	for _, item := range courses {
		publicCourses = append(publicCourses, ToPublicCourseDTO(&item))
	}

	return c.JSON(http.StatusOK, publicCourses)
}

// ListAdminCourses handles GET /api/v1/admin/courses (all courses across all authors).
func (r *CourseRouter) ListAdminCourses(c *echo.Context) error {
	status := entities.ContentStatus(c.QueryParam("status"))
	switch status {
	case "", entities.ContentStatusDraft, entities.ContentStatusInReview,
		entities.ContentStatusPublished, entities.ContentStatusArchived:
	default:
		return response.Fail(c, http.StatusBadRequest, "El estado de curso solicitado no es válido.")
	}
	ownerType := entities.OwnerType(c.QueryParam("owner_type"))
	if ownerType != "" && !ownerType.IsValid() {
		return response.Fail(c, http.StatusBadRequest, "El tipo de propietario solicitado no es válido.")
	}
	courses, err := r.listAdminCoursesHandler.Handle(c.Request().Context(), courseQuery.ListAdminCoursesQuery{
		Status:    status,
		OwnerType: ownerType,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la lista administrativa de cursos.", err)
	}

	return c.JSON(http.StatusOK, courses)
}
