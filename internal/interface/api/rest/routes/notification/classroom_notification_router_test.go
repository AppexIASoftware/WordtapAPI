package notification_test

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
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/notification"
)

type mockClassroomNotifRepoForAdminRouter struct {
	notifications []entities.ClassroomNotification
}

func (m *mockClassroomNotifRepoForAdminRouter) Create(context.Context, *entities.ClassroomNotification) error {
	return nil
}
func (m *mockClassroomNotifRepoForAdminRouter) ListByClassroom(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForAdminRouter) ListByTeacher(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForAdminRouter) ListByCourse(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForAdminRouter) ListByLesson(context.Context, string, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForAdminRouter) ListAll(context.Context, domainRepo.ClassroomNotificationFilter, int, int) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForAdminRouter) CountByClassroom(context.Context, string) (int64, error) {
	return int64(len(m.notifications)), nil
}
func (m *mockClassroomNotifRepoForAdminRouter) CountByTeacher(context.Context, string) (int64, error) {
	return int64(len(m.notifications)), nil
}
func (m *mockClassroomNotifRepoForAdminRouter) CountAll(context.Context, domainRepo.ClassroomNotificationFilter) (int64, error) {
	return int64(len(m.notifications)), nil
}
func (m *mockClassroomNotifRepoForAdminRouter) ListClassroomCoursesForTeacher(context.Context, string) ([]string, error) {
	return []string{}, nil
}
func (m *mockClassroomNotifRepoForAdminRouter) ListByClassroomWithCourseScope(context.Context, string, int, int, ...domainRepo.ClassroomNotificationFilter) ([]entities.ClassroomNotification, error) {
	return m.notifications, nil
}
func (m *mockClassroomNotifRepoForAdminRouter) CountByClassroomWithCourseScope(context.Context, string, ...domainRepo.ClassroomNotificationFilter) (int64, error) {
	return int64(len(m.notifications)), nil
}

type mockClassroomRepoForAdminRouter struct{}

func (m *mockClassroomRepoForAdminRouter) Create(context.Context, *entities.Classroom) error {
	return nil
}
func (m *mockClassroomRepoForAdminRouter) FindByID(context.Context, string) (*entities.Classroom, error) {
	return &entities.Classroom{ID: "c-1"}, nil
}
func (m *mockClassroomRepoForAdminRouter) FindByCode(context.Context, string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForAdminRouter) FindByTeacher(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForAdminRouter) FindByStudent(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForAdminRouter) EnrollStudent(context.Context, *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockClassroomRepoForAdminRouter) FindEnrollment(context.Context, string, string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomRepoForAdminRouter) ListEnrollments(context.Context, string) ([]entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomRepoForAdminRouter) ListStumbles(context.Context, string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockClassroomRepoForAdminRouter) CreateStumble(context.Context, *entities.StudentStumble) error {
	return nil
}
func (m *mockClassroomRepoForAdminRouter) Update(context.Context, *entities.Classroom) error {
	return nil
}

func TestClassroomNotificationRouter_List(t *testing.T) {
	e := echo.New()
	now := time.Now()
	notifs := []entities.ClassroomNotification{
		{
			ID:      "admin-notif-1",
			Title:   "Global Announcement",
			Message: "Platform update",
			SentAt:  now,
		},
	}

	handler := queries.NewListClassroomNotificationsHandler(
		&mockClassroomNotifRepoForAdminRouter{notifications: notifs},
		&mockClassroomRepoForAdminRouter{},
	)
	router := notification.NewClassroomNotificationRouter(handler)
	noopMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(e.Group("/api/v1"), noopMiddleware, noopMiddleware)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/notifications?limit=15&offset=0", nil)
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
	if len(res.Notifications) != 1 || res.Notifications[0].ID != "admin-notif-1" {
		t.Fatalf("unexpected data: %+v", res.Notifications)
	}
	if res.Pagination.Total != 1 {
		t.Fatalf("expected total 1, got %d", res.Pagination.Total)
	}
}
