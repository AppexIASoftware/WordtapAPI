package classroom

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type ClassroomRouter struct {
	createClassroomHandler *commands.CreateClassroomHandler
	updateClassroomHandler *commands.UpdateClassroomHandler
	enrollStudentHandler   *commands.EnrollStudentHandler
	listClassroomsHandler  *queries.ListTeacherClassroomsHandler
	getStumbleListHandler  *queries.GetStumbleListHandler
}

func NewClassroomRouter(
	createClassroomHandler *commands.CreateClassroomHandler,
	updateClassroomHandler *commands.UpdateClassroomHandler,
	enrollStudentHandler *commands.EnrollStudentHandler,
	listClassroomsHandler *queries.ListTeacherClassroomsHandler,
	getStumbleListHandler *queries.GetStumbleListHandler,
) *ClassroomRouter {
	return &ClassroomRouter{
		createClassroomHandler: createClassroomHandler,
		updateClassroomHandler: updateClassroomHandler,
		enrollStudentHandler:   enrollStudentHandler,
		listClassroomsHandler:  listClassroomsHandler,
		getStumbleListHandler:  getStumbleListHandler,
	}
}

func (r *ClassroomRouter) RegisterRoutes(
	v1 *echo.Group,
	authRequired echo.MiddlewareFunc,
	instructorOrAdmin echo.MiddlewareFunc,
) {
	// Teacher classroom management routes
	teacherGroup := v1.Group("/teacher/classrooms")
	teacherGroup.Use(authRequired, instructorOrAdmin)
	teacherGroup.POST("", r.CreateClassroom)
	teacherGroup.PUT("/:id", r.UpdateClassroom)
	teacherGroup.GET("", r.ListTeacherClassrooms)
	teacherGroup.GET("/:id/stumbles", r.GetClassroomStumbles)
}

func (r *ClassroomRouter) CreateClassroom(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	var req CreateClassroomRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	cmd := commands.CreateClassroomCommand{
		TeacherID:       userID,
		Name:            req.Name,
		Level:           req.Level,
		CourseID:        req.CourseID,
		CourseIDs:       req.CourseIDs,
		LessonIDs:       req.LessonIDs,
		GameIDs:         req.GameIDs,
		WeeklyGoalWords: req.WeeklyGoalWords,
		Description:     req.Description,
	}

	classroom, err := r.createClassroomHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo crear el aula.", err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"success": true,
		"data":    formatClassroomDTO(classroom),
	})
}

func (r *ClassroomRouter) UpdateClassroom(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	classroomID := c.Param("id")
	if classroomID == "" {
		return response.Fail(c, http.StatusBadRequest, "Falta el identificador del aula.")
	}

	var req UpdateClassroomRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Los datos enviados no son válidos.", err)
	}

	cmd := commands.UpdateClassroomCommand{
		ClassroomID:     classroomID,
		TeacherID:       userID,
		Name:            req.Name,
		Level:           req.Level,
		CourseID:        req.CourseID,
		CourseIDs:       req.CourseIDs,
		LessonIDs:       req.LessonIDs,
		GameIDs:         req.GameIDs,
		WeeklyGoalWords: req.WeeklyGoalWords,
		Description:     req.Description,
	}

	classroom, err := r.updateClassroomHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "No se pudo actualizar el aula.", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    formatClassroomDTO(classroom),
	})
}

func (r *ClassroomRouter) ListTeacherClassrooms(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}

	classrooms, err := r.listClassroomsHandler.Handle(c.Request().Context(), queries.ListTeacherClassroomsQuery{
		TeacherID: userID,
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron cargar las aulas.", err)
	}

	payload := make([]map[string]any, 0, len(classrooms))
	for _, cls := range classrooms {
		payload = append(payload, formatClassroomListDTO(&cls))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    payload,
	})
}
