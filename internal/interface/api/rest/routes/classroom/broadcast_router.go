package classroom

import (
	"net/http"

	"github.com/labstack/echo/v5"

	classroomCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

type BroadcastRouter struct {
	handler *classroomCmd.BroadcastClassroomHandler
}

func NewBroadcastRouter(handler *classroomCmd.BroadcastClassroomHandler) *BroadcastRouter {
	return &BroadcastRouter{handler: handler}
}

func (r *BroadcastRouter) RegisterRoutes(v1 *echo.Group, authRequired, instructorOrAdmin echo.MiddlewareFunc) {
	group := v1.Group("/teacher/classrooms")
	group.Use(authRequired, instructorOrAdmin)
	group.POST("/:id/broadcast", r.Broadcast)
}

func (r *BroadcastRouter) Broadcast(c *echo.Context) error {
	teacherID, ok := c.Get("user_id").(string)
	if !ok || teacherID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Autenticación requerida.")
	}

	classroomID := c.Param("id")
	var body struct {
		Title         string `json:"title"`
		Message       string `json:"message"`
		ActionType    string `json:"action_type"`
		ActionPayload string `json:"action_payload"`
	}
	if err := c.Bind(&body); err != nil {
		return response.Fail(c, http.StatusBadRequest, "Datos de entrada inválidos.")
	}

	cmd := classroomCmd.BroadcastClassroomCommand{
		ClassroomID:   classroomID,
		TeacherID:     teacherID,
		Title:         body.Title,
		Message:       body.Message,
		ActionType:    body.ActionType,
		ActionPayload: body.ActionPayload,
	}

	res, err := r.handler.Handle(c.Request().Context(), cmd)
	if err != nil {
		if err == classroomCmd.ErrForbiddenTeacher {
			return response.Fail(c, http.StatusForbidden, err.Error())
		}
		if err == classroomCmd.ErrClassroomNotFound {
			return response.Fail(c, http.StatusNotFound, err.Error())
		}
		return response.Fail(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, res)
}
