package gamification_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/access"
	gameQuery "github.com/AppexIASoftware/WordtapAPI/internal/application/features/gamification/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/gamification"
)

type routerGameLinks struct {
	links map[string][]entities.LessonGame
	err   error
}

func (r *routerGameLinks) FindLessonsByGame(_ context.Context, id string) ([]entities.LessonGame, error) {
	return r.links[id], r.err
}

type routerPurchases struct{}

func (routerPurchases) HasPurchase(_ context.Context, student, course string) (bool, error) {
	return student == "buyer" && course == "course", nil
}

func TestStudentGameRoutesEnforceLinkedCourseAccess(t *testing.T) {
	course := &entities.Course{ID: "course", OwnerType: entities.OwnerTypeTeacher, Status: entities.ContentStatusPublished}
	lesson := &entities.Lesson{ID: "lesson", CourseID: course.ID, Course: course, Status: entities.ContentStatusPublished}
	links := &routerGameLinks{links: map[string][]entities.LessonGame{
		"linked": {{TeacherGameID: "linked", LessonID: lesson.ID, Lesson: lesson}},
	}}
	classrooms := &mockStudentClassroomRepoForRouter{classrooms: []entities.Classroom{{
		ID: "classroom", TeacherID: "teacher", IsActive: true,
		Enrollments: []entities.ClassroomEnrollment{{StudentID: "buyer"}, {StudentID: "other"}},
	}}}
	games := newMockGameRouterRepo()
	for _, id := range []string{"linked", "independent"} {
		games.games = append(games.games, entities.TeacherGame{ID: id, TeacherID: "teacher", IsActive: true, IsPublishedStandalone: true, AudienceScope: "all_students", SourceType: "custom_list"})
	}
	guard := gameQuery.NewCourseGameAccess(nil, links, access.NewContentPolicy(routerPurchases{}, true))
	list := gameQuery.NewListStudentTeacherGamesHandler(classrooms, games).WithCourseGameAccess(guard)
	get := gameQuery.NewGetStudentTeacherGameDetailHandler(classrooms, games).WithCourseGameAccess(guard)
	e := echo.New()
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", c.Request().Header.Get("X-Student"))
			return next(c)
		}
	}
	gamification.NewGameStudentRouter(list, get).RegisterRoutes(e.Group("/api/v1"), auth)
	request := func(student, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Student", student)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	for _, tc := range []struct {
		student, game string
		status        int
	}{
		{"other", "linked", http.StatusNotFound},
		{"buyer", "linked", http.StatusOK},
		{"other", "independent", http.StatusOK},
		{"outsider", "independent", http.StatusNotFound},
	} {
		t.Run(tc.student+"/"+tc.game, func(t *testing.T) {
			if rec := request(tc.student, "/api/v1/games/student/"+tc.game); rec.Code != tc.status {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
	for _, tc := range []struct {
		student string
		count   int
	}{{"other", 1}, {"buyer", 2}, {"outsider", 0}} {
		rec := request(tc.student, "/api/v1/games/student")
		var body struct {
			Data []map[string]any `json:"data"`
		}
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil || len(body.Data) != tc.count {
			t.Fatalf("%s list: %d %s", tc.student, rec.Code, rec.Body.String())
		}
	}
	links.err = errors.New("database unavailable")
	if rec := request("buyer", "/api/v1/games/student"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected fail-closed list, got %d", rec.Code)
	}
	if rec := request("buyer", "/api/v1/games/student/linked"); rec.Code != http.StatusNotFound {
		t.Fatalf("expected fail-closed detail, got %d", rec.Code)
	}
}
