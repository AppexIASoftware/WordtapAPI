package course

import (
	"context"
	"errors"
	"net/http"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
	"github.com/labstack/echo/v5"
)

type CourseAccessRouter struct {
	handler     *commands.GrantTestCourseAccessHandler
	reader      testAccessReader
	testEnabled bool
}

type testAccessReader interface {
	HasTestAccess(context.Context, string, string, bool) (bool, error)
}

func (r *CourseAccessRouter) WithAccessReader(reader testAccessReader, enabled bool) *CourseAccessRouter {
	r.reader, r.testEnabled = reader, enabled
	return r
}

func NewCourseAccessRouter(handler *commands.GrantTestCourseAccessHandler) *CourseAccessRouter {
	return &CourseAccessRouter{handler: handler}
}

func (r *CourseAccessRouter) RegisterRoutes(v1 *echo.Group, auth, student echo.MiddlewareFunc) {
	v1.POST("/student/courses/:id/test-access", r.GrantTestAccess, auth, student)
	v1.GET("/student/courses/:id/test-access", r.FindTestAccess, auth, student)
}

func (r *CourseAccessRouter) FindTestAccess(c *echo.Context) error {
	if !r.testEnabled || r.reader == nil {
		return response.Fail(c, http.StatusNotFound, "Access is not available.")
	}
	userID, _ := c.Get("user_id").(string)
	if userID == "" || c.Param("id") == "" {
		return response.Fail(c, http.StatusBadRequest, "Unable to check course access.")
	}
	hasAccess, err := r.reader.HasTestAccess(c.Request().Context(), userID, c.Param("id"), r.testEnabled)
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, "Unable to check course access.")
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true, "has_access": hasAccess})
}

func (r *CourseAccessRouter) GrantTestAccess(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	err := r.handler.Handle(c.Request().Context(), userID, c.Param("id"))
	switch {
	case errors.Is(err, commands.ErrTestAccessDisabled):
		return response.Fail(c, http.StatusNotFound, "Access is not available.")
	case errors.Is(err, commands.ErrInvalidCourseAccess):
		return response.Fail(c, http.StatusBadRequest, "Unable to grant course access.")
	case err != nil:
		return response.Fail(c, http.StatusInternalServerError, "Unable to grant course access.")
	default:
		return c.JSON(http.StatusOK, map[string]any{"success": true})
	}
}
