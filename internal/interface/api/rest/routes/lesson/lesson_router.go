package lesson

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/lesson/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type LessonRouter struct {
	getLessonDetailHandler   *queries.GetLessonDetailHandler
	listCourseLessonsHandler *queries.ListCourseLessonsHandler
	createLessonHandler      *commands.CreateLessonHandler
	updateLessonHandler      *commands.UpdateLessonHandler
	deleteLessonHandler      *commands.DeleteLessonHandler
	saveLessonItemHandler    *commands.SaveLessonItemHandler
	deleteLessonItemHandler  *commands.DeleteLessonItemHandler
}

func NewLessonRouter(
	lessonRepo repositories.LessonRepository,
	listCourseLessonsHandler *queries.ListCourseLessonsHandler,
	createLessonHandler *commands.CreateLessonHandler,
	updateLessonHandler *commands.UpdateLessonHandler,
	deleteLessonHandler *commands.DeleteLessonHandler,
	saveLessonItemHandler *commands.SaveLessonItemHandler,
	deleteLessonItemHandler *commands.DeleteLessonItemHandler,
) *LessonRouter {
	return &LessonRouter{
		getLessonDetailHandler:   queries.NewGetLessonDetailHandler(lessonRepo),
		listCourseLessonsHandler: listCourseLessonsHandler,
		createLessonHandler:      createLessonHandler,
		updateLessonHandler:      updateLessonHandler,
		deleteLessonHandler:      deleteLessonHandler,
		saveLessonItemHandler:    saveLessonItemHandler,
		deleteLessonItemHandler:  deleteLessonItemHandler,
	}
}

func (r *LessonRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	studentOrStaff echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
) {
	// Course lessons (students and staff: instructor, admin, moderator)
	v1.GET("/courses/:course_id/lessons", r.ListCourseLessons, authRequired, studentOrStaff)
	v1.POST("/courses/:course_id/lessons", r.CreateLesson, authRequired, instructorOrAdmin)

	// Single lesson management
	lessonsGroup := v1.Group("/lessons")
	lessonsGroup.GET("/:id", r.GetLessonDetail, authRequired, studentOrStaff)
	lessonsGroup.PUT("/:id", r.UpdateLesson, authRequired, instructorOrAdmin)
	lessonsGroup.DELETE("/:id", r.DeleteLesson, authRequired, instructorOrAdmin)
	lessonsGroup.POST("/:id/items", r.SaveLessonItem, authRequired, instructorOrAdmin)
	lessonsGroup.DELETE("/:id/items/:item_id", r.DeleteLessonItem, authRequired, instructorOrAdmin)
}

// GetLessonDetail handles GET /api/v1/lessons/:id
func (r *LessonRouter) GetLessonDetail(c *echo.Context) error {
	identifier := c.Param("id")
	if identifier == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la lección.")
	}

	ctx := c.Request().Context()
	dto, err := r.getLessonDetailHandler.Handle(ctx, queries.GetLessonDetailQuery{Identifier: identifier})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Fail(c, http.StatusNotFound, "No se encontró la lección.")
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudo cargar la lección.", err)
	}

	return c.JSON(http.StatusOK, dto)
}

// ListCourseLessons handles GET /api/v1/courses/:course_id/lessons
func (r *LessonRouter) ListCourseLessons(c *echo.Context) error {
	courseID := c.Param("course_id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	userID, _ := c.Get("user_id").(string)
	roleStr, _ := c.Get("user_role").(string)

	ctx := c.Request().Context()
	lessons, err := r.listCourseLessonsHandler.Handle(ctx, queries.ListCourseLessonsQuery{
		CourseID:      courseID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		if errors.Is(err, queries.ErrCourseLessonsAccessDenied) {
			if userID == "" {
				return response.Fail(c, http.StatusNotFound, "No se encontró el curso.")
			}
			return response.Fail(c, http.StatusForbidden, "No tienes permisos para ver las lecciones de este curso.")
		}
		if errors.Is(err, queries.ErrCourseLessonsNotFound) {
			return response.Fail(c, http.StatusNotFound, "No se encontró el curso.")
		}
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar las lecciones.", err)
	}

	return c.JSON(http.StatusOK, lessons)
}

type CreateLessonRequest struct {
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	EstimatedMinutes *int    `json:"estimated_minutes"`
	SortOrder        int     `json:"sort_order"`
}

// CreateLesson handles POST /api/v1/courses/:course_id/lessons
func (r *LessonRouter) CreateLesson(c *echo.Context) error {
	courseID := c.Param("course_id")
	if courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del curso.")
	}

	var req CreateLessonRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	lesson, err := r.createLessonHandler.Handle(ctx, commands.CreateLessonCommand{
		CourseID:         courseID,
		Title:            req.Title,
		Description:      req.Description,
		EstimatedMinutes: req.EstimatedMinutes,
		SortOrder:        req.SortOrder,
		RequesterID:      userID,
		RequesterRole:    entities.UserRole(roleStr),
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo crear la lección.", err)
	}

	return c.JSON(http.StatusCreated, lesson)
}

type UpdateLessonRequest struct {
	Title            *string `json:"title"`
	Description      *string `json:"description"`
	EstimatedMinutes *int    `json:"estimated_minutes"`
	SortOrder        *int    `json:"sort_order"`
}

// UpdateLesson handles PUT /api/v1/lessons/:id
func (r *LessonRouter) UpdateLesson(c *echo.Context) error {
	lessonID := c.Param("id")
	if lessonID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la lección.")
	}

	var req UpdateLessonRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	lesson, err := r.updateLessonHandler.Handle(ctx, commands.UpdateLessonCommand{
		LessonID:         lessonID,
		Title:            req.Title,
		Description:      req.Description,
		EstimatedMinutes: req.EstimatedMinutes,
		SortOrder:        req.SortOrder,
		RequesterID:      userID,
		RequesterRole:    entities.UserRole(roleStr),
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar la lección.", err)
	}

	return c.JSON(http.StatusOK, lesson)
}

// DeleteLesson handles DELETE /api/v1/lessons/:id
func (r *LessonRouter) DeleteLesson(c *echo.Context) error {
	lessonID := c.Param("id")
	if lessonID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la lección.")
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	err := r.deleteLessonHandler.Handle(ctx, commands.DeleteLessonCommand{
		LessonID:      lessonID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar la lección.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "deleted", "lesson_id": lessonID})
}

type SaveLessonItemRequest struct {
	ID                string  `json:"id"`
	ContentBankItemID *string `json:"content_bank_item_id"`
	ItemType          string  `json:"item_type"`
	ContentText       *string `json:"content_text"`
	SortOrder         int     `json:"sort_order"`
	IsRequired        bool    `json:"is_required"`
}

// SaveLessonItem handles POST /api/v1/lessons/:id/items
func (r *LessonRouter) SaveLessonItem(c *echo.Context) error {
	lessonID := c.Param("id")
	if lessonID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la lección.")
	}

	var req SaveLessonItemRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	item, err := r.saveLessonItemHandler.Handle(ctx, commands.SaveLessonItemCommand{
		ItemID:            req.ID,
		LessonID:          lessonID,
		ContentBankItemID: req.ContentBankItemID,
		ItemType:          entities.LessonContentType(req.ItemType),
		ContentText:       req.ContentText,
		SortOrder:         req.SortOrder,
		IsRequired:        req.IsRequired,
		RequesterID:       userID,
		RequesterRole:     entities.UserRole(roleStr),
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo guardar la tarjeta de la lección.", err)
	}

	return c.JSON(http.StatusOK, item)
}

// DeleteLessonItem handles DELETE /api/v1/lessons/:id/items/:item_id
func (r *LessonRouter) DeleteLessonItem(c *echo.Context) error {
	lessonID := c.Param("id")
	itemID := c.Param("item_id")
	if lessonID == "" || itemID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador de la lección o de la tarjeta.")
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	err := r.deleteLessonItemHandler.Handle(ctx, commands.DeleteLessonItemCommand{
		LessonID:      lessonID,
		ItemID:        itemID,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo eliminar la tarjeta de la lección.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "deleted", "item_id": itemID})
}
