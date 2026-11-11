package classroom_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	classroomCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/classroom"
)

type mockClassroomRepoForBroadcastRoute struct {
	classroom   *entities.Classroom
	enrollments []entities.ClassroomEnrollment
}

func (m *mockClassroomRepoForBroadcastRoute) Create(context.Context, *entities.Classroom) error {
	return nil
}
func (m *mockClassroomRepoForBroadcastRoute) FindByID(context.Context, string) (*entities.Classroom, error) {
	return m.classroom, nil
}
func (m *mockClassroomRepoForBroadcastRoute) FindByCode(context.Context, string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcastRoute) FindByTeacher(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcastRoute) FindByStudent(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcastRoute) EnrollStudent(context.Context, *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockClassroomRepoForBroadcastRoute) FindEnrollment(context.Context, string, string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcastRoute) ListEnrollments(context.Context, string) ([]entities.ClassroomEnrollment, error) {
	return m.enrollments, nil
}
func (m *mockClassroomRepoForBroadcastRoute) ListStumbles(context.Context, string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcastRoute) CreateStumble(context.Context, *entities.StudentStumble) error {
	return nil
}
func (m *mockClassroomRepoForBroadcastRoute) Update(context.Context, *entities.Classroom) error {
	return nil
}

type mockInboxRepoForBroadcastRoute struct{}

func (m *mockInboxRepoForBroadcastRoute) List(context.Context, string, int, *inbox.Cursor) (inbox.Page, error) {
	return inbox.Page{}, nil
}
func (m *mockInboxRepoForBroadcastRoute) UnreadCount(context.Context, string) (int64, error) {
	return 0, nil
}
func (m *mockInboxRepoForBroadcastRoute) MarkRead(context.Context, string, string) (bool, error) {
	return true, nil
}
func (m *mockInboxRepoForBroadcastRoute) Create(context.Context, *entities.Notification) error {
	return nil
}
func (m *mockInboxRepoForBroadcastRoute) SaveBroadcast(context.Context, *entities.ClassroomNotification, []entities.Notification) error {
	return nil
}
func (m *mockInboxRepoForBroadcastRoute) CreateBatch(context.Context, []entities.Notification) error {
	return nil
}

func (m *mockInboxRepoForBroadcastRoute) UpdateStatus(context.Context, string, entities.ClassroomNotificationStatus) error {
	return nil
}

type mockDeviceRepoForBroadcastRoute struct{}

func (m *mockDeviceRepoForBroadcastRoute) RegisterDevice(context.Context, *entities.UserDevice) error {
	return nil
}
func (m *mockDeviceRepoForBroadcastRoute) RevokeToken(context.Context, string, string) error {
	return nil
}
func (m *mockDeviceRepoForBroadcastRoute) GetTokensByUser(context.Context, string) ([]string, error) {
	return []string{"tok-1"}, nil
}
func (m *mockDeviceRepoForBroadcastRoute) GetTokensByUsers(context.Context, []string) ([]string, error) {
	return []string{"tok-1"}, nil
}
func (m *mockDeviceRepoForBroadcastRoute) GetAllTokens(context.Context) ([]string, error) {
	return []string{"tok-1"}, nil
}
func (m *mockDeviceRepoForBroadcastRoute) CountActiveTokens(context.Context) (int64, error) {
	return 1, nil
}

func TestBroadcastRouter_Broadcast(t *testing.T) {
	clsRepo := &mockClassroomRepoForBroadcastRoute{
		classroom: &entities.Classroom{
			ID:        "cls-123",
			TeacherID: "teacher-1",
		},
		enrollments: []entities.ClassroomEnrollment{
			{StudentID: "student-1"},
		},
	}
	inboxRepo := &mockInboxRepoForBroadcastRoute{}
	deviceRepo := &mockDeviceRepoForBroadcastRoute{}

	handler := classroomCmd.NewBroadcastClassroomHandler(clsRepo, inboxRepo, deviceRepo, nil)
	router := classroom.NewBroadcastRouter(handler)

	e := echo.New()
	v1 := e.Group("/api/v1")
	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user_id", "teacher-1")
			return next(c)
		}
	}
	passMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	router.RegisterRoutes(v1, authMiddleware, passMiddleware)

	body, _ := json.Marshal(map[string]any{
		"title":          "Práctica de verbos",
		"message":        "Realizar juego 1",
		"action_type":    "open_lesson",
		"action_payload": "les-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/classrooms/cls-123/broadcast", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
}
