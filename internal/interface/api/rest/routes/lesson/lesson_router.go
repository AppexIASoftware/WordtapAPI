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
	staffRoles echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
) {
	// Course lessons (staff only: instructor, admin, moderator)
	v1.GET("/courses/:course_id/lessons", r.ListCourseLessons, authRequired, staffRoles)
	v1.POST("/courses/:course_id/lessons", r.CreateLesson, authRequired, instructorOrAdmin)

	// Single lesson management
	lessonsGroup := v1.Group("/lessons")
	lessonsGroup.GET("/:id", r.GetLessonDetail, authRequired, staffRoles)
	lessonsGroup.PUT("/:id", r.UpdateLesson, authRequired, instructorOrAdmin)
	lessonsGroup.DELETE("/:id", r.DeleteLesson, authRequired, instructorOrAdmin)
	lessonsGroup.POST("/:id/items", r.SaveLessonItem, authRequired, instructorOrAdmin)
	lessonsGroup.DELETE("/:id/items/:item_id", r.DeleteLessonItem, authRequired, instructorOrAdmin)
}

// GetLessonDetail handles GET /api/v1/lessons/:id
func (r *LessonRouter) GetLessonDetail(c *echo.Context) error {
	identifier := c.Param("id")
	if identifier == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "missing lesson identifier"})
	}

	ctx := c.Request().Context()
	dto, err := r.getLessonDetailHandler.Handle(ctx, queries.GetLessonDetailQuery{Identifier: identifier})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]any{"error": "lesson not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to fetch lesson detail"})
	}

	return c.JSON(http.StatusOK, dto)
}

// ListCourseLessons handles GET /api/v1/courses/:course_id/lessons
func (r *LessonRouter) ListCourseLessons(c *echo.Context) error {
	courseID := c.Param("course_id")
	if courseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "missing course id parameter"})
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
				return c.JSON(http.StatusNotFound, map[string]any{"error": "course not found"})
			}
			return c.JSON(http.StatusForbidden, map[string]any{"error": "access denied: course is not published"})
		}
		if errors.Is(err, queries.ErrCourseLessonsNotFound) {
			return c.JSON(http.StatusNotFound, map[string]any{"error": "course not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "failed to fetch course lessons",
			"details": err.Error(),
		})
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
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "missing course id parameter"})
	}

	var req CreateLessonRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body", "details": err.Error()})
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
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to create lesson", "details": err.Error()})
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
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "missing lesson id parameter"})
	}

	var req UpdateLessonRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body", "details": err.Error()})
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
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to update lesson", "details": err.Error()})
	}

	return c.JSON(http.StatusOK, lesson)
}

// DeleteLesson handles DELETE /api/v1/lessons/:id
func (r *LessonRouter) DeleteLesson(c *echo.Context) error {
	lessonID := c.Param("id")
	if lessonID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "missing lesson id parameter"})
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
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to delete lesson", "details": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "deleted", "lesson_id": lessonID})
}

type SaveLessonItemRequest struct {
	ID          string  `json:"id"`
	ItemType    string  `json:"item_type"`
	ContentText *string `json:"content_text"`
	SortOrder   int     `json:"sort_order"`
	IsRequired  bool    `json:"is_required"`
}

// SaveLessonItem handles POST /api/v1/lessons/:id/items
func (r *LessonRouter) SaveLessonItem(c *echo.Context) error {
	lessonID := c.Param("id")
	if lessonID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "missing lesson id parameter"})
	}

	var req SaveLessonItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body", "details": err.Error()})
	}

	userIDVal := c.Get("user_id")
	userID, _ := userIDVal.(string)

	roleVal := c.Get("user_role")
	roleStr, _ := roleVal.(string)

	ctx := c.Request().Context()
	item, err := r.saveLessonItemHandler.Handle(ctx, commands.SaveLessonItemCommand{
		ItemID:        req.ID,
		LessonID:      lessonID,
		ItemType:      entities.LessonContentType(req.ItemType),
		ContentText:   req.ContentText,
		SortOrder:     req.SortOrder,
		IsRequired:    req.IsRequired,
		RequesterID:   userID,
		RequesterRole: entities.UserRole(roleStr),
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to save lesson card", "details": err.Error()})
	}

	return c.JSON(http.StatusOK, item)
}

// DeleteLessonItem handles DELETE /api/v1/lessons/:id/items/:item_id
func (r *LessonRouter) DeleteLessonItem(c *echo.Context) error {
	lessonID := c.Param("id")
	itemID := c.Param("item_id")
	if lessonID == "" || itemID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "missing lesson or item id parameter"})
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
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "failed to delete lesson card", "details": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "deleted", "item_id": itemID})
}
