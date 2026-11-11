package classroom_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/queries"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/classroom"
)

type mockClassroomNotifRepoForRouter struct {
	notifications []entities.ClassroomNotification
}

func (m *mockClassroomNotifRepoForRouter) Create(context.Context, *entities.ClassroomNotification) error {
	return nil
}
func (m *mockClassroomNotifRepoForRouter) ListByClassroom(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForRouter) ListByTeacher(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForRouter) ListByCourse(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForRouter) ListByLesson(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForRouter) ListAll(context.Context, domainRepo.ClassroomNotificationFilter, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForRouter) CountByClassroom(context.Context, string) (int64, error) {
	return int64(len(m.notifications)), nil
}
func (m *mockClassroomNotifRepoForRouter) CountByTeacher(context.Context, string) (int64, error) {
	return int64(len(m.notifications)), nil
}
func (m *mockClassroomNotifRepoForRouter) CountAll(context.Context, domainRepo.ClassroomNotificationFilter) (int64, error) {
	return int64(len(m.notifications)), nil
}
func (m *mockClassroomNotifRepoForRouter) ListClassroomCoursesForTeacher(context.Context, string) ([]string, error) {
	return []string{"course-1"}, nil
}
func (m *mockClassroomNotifRepoForRouter) ListByClassroomWithCourseScope(context.Context, string, int, int, ...domainRepo.ClassroomNotificationFilter) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForRouter) CountByClassroomWithCourseScope(context.Context, string, ...domainRepo.ClassroomNotificationFilter) (int64, error) {
	return int64(len(m.notifications)), nil
}

func TestNotificationRouter_List_Unauthorized(t *testing.T) {
	e := echo.New()
	handler := queries.NewListClassroomNotificationsHandler(&mockClassroomNotifRepoForRouter{}, &mockClassroomRepoForBroadcastRoute{})
	router := classroom.NewNotificationRouter(handler)
	noopMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(e.Group("/api/v1"), noopMiddleware, noopMiddleware)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/classrooms/c-1/notifications", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestNotificationRouter_List_Success(t *testing.T) {
	e := echo.New()
	now := time.Now()
	cid := "c-1"
	notifs := []entities.ClassroomNotification{
		{
			ID:          "notif-1",
			ClassroomID: &cid,
			Title:       "Test Notification",
			Message:     "Hello students",
			SentAt:      now,
		},
	}

	crMock := &mockClassroomRepoForBroadcastRoute{
		classroom: &entities.Classroom{
			ID:        "c-1",
			TeacherID: "teacher-123",
		},
	}
	handler := queries.NewListClassroomNotificationsHandler(&mockClassroomNotifRepoForRouter{notifications: notifs}, crMock)
	router := classroom.NewNotificationRouter(handler)

	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-123")
			return next(c)
		}
	}
	noopMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(e.Group("/api/v1"), authMiddleware, noopMiddleware)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/classrooms/c-1/notifications?limit=10&offset=0", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var res struct {
		Notifications []entities.ClassroomNotification `json:"notifications"`
		Pagination    struct {
			Total  int64 `json:"total"`
			Limit  int   `json:"limit"`
			Offset int   `json:"offset"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(res.Notifications) != 1 || res.Notifications[0].ID != "notif-1" {
		t.Fatalf("unexpected notification list: %+v", res.Notifications)
	}
	if res.Pagination.Total != 1 {
		t.Fatalf("expected total 1, got %d", res.Pagination.Total)
	}
}
