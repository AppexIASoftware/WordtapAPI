package classroom_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	commands "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	queries "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/classroom"
	"github.com/labstack/echo/v5"
)

type selectionCourseLookup struct{}

func (selectionCourseLookup) FindByID(_ context.Context, id string) (*entities.Course, error) {
	return &entities.Course{ID: id, OwnerType: entities.OwnerTypePlatform, Status: entities.ContentStatusPublished,
		Lessons: []entities.Lesson{{ID: "lesson-1", CourseID: id, Status: entities.ContentStatusPublished}}}, nil
}

func TestClassroomSelectionHTTP(t *testing.T) {
	repo := newMockClassroomRepo()
	e := echo.New()
	r := classroom.NewClassroomRouter(commands.NewCreateClassroomHandler(repo, selectionCourseLookup{}), commands.NewUpdateClassroomHandler(repo, selectionCourseLookup{}), commands.NewEnrollStudentHandler(repo), queries.NewListTeacherClassroomsHandler(repo), queries.NewGetStumbleListHandler(repo))
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error { c.Set("user_id", "teacher"); return next(c) }
	}
	r.RegisterRoutes(e.Group("/api/v1"), auth, auth)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	created := call(http.MethodPost, "/api/v1/teacher/classrooms", `{"name":"Class","course_ids":["course"],"lesson_ids":["lesson-1"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	if len(repo.classrooms) != 1 || repo.classrooms[0].AssignedLessons == nil || *repo.classrooms[0].AssignedLessons != `["lesson-1"]` {
		t.Fatalf("stored: %+v", repo.classrooms)
	}
	listed := call(http.MethodGet, "/api/v1/teacher/classrooms", "")
	var result struct {
		Data []struct {
			Lessons []string `json:"assignedLessons"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &result); err != nil || len(result.Data) != 1 || len(result.Data[0].Lessons) != 1 || result.Data[0].Lessons[0] != "lesson-1" {
		t.Fatalf("list: %s, %v", listed.Body, err)
	}
	repo.classrooms[0].ID = "class"
	updated := call(http.MethodPut, "/api/v1/teacher/classrooms/class", `{"name":"Class","lesson_ids":[]}`)
	if updated.Code != http.StatusOK || *repo.classrooms[0].AssignedLessons != `[]` {
		t.Fatalf("update: %d %s", updated.Code, updated.Body)
	}
	invalid := call(http.MethodPut, "/api/v1/teacher/classrooms/class", `{"name":"Class","lesson_ids":["unknown"]}`)
	if invalid.Code != http.StatusBadRequest || *repo.classrooms[0].AssignedLessons != `[]` {
		t.Fatalf("invalid: %d %s", invalid.Code, invalid.Body)
	}
}
