package commands_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/classroom/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockClassroomRepoForBroadcast struct {
	classroom   *entities.Classroom
	enrollments []entities.ClassroomEnrollment
	err         error
}

func (m *mockClassroomRepoForBroadcast) Create(context.Context, *entities.Classroom) error {
	return nil
}
func (m *mockClassroomRepoForBroadcast) FindByID(context.Context, string) (*entities.Classroom, error) {
	return m.classroom, m.err
}
func (m *mockClassroomRepoForBroadcast) FindByCode(context.Context, string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcast) FindByTeacher(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcast) FindByStudent(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcast) EnrollStudent(context.Context, *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockClassroomRepoForBroadcast) FindEnrollment(context.Context, string, string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcast) ListEnrollments(context.Context, string) ([]entities.ClassroomEnrollment, error) {
	return m.enrollments, m.err
}
func (m *mockClassroomRepoForBroadcast) ListStumbles(context.Context, string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockClassroomRepoForBroadcast) CreateStumble(context.Context, *entities.StudentStumble) error {
	return nil
}
func (m *mockClassroomRepoForBroadcast) Update(context.Context, *entities.Classroom) error {
	return nil
}

type mockInboxRepoForBroadcast struct {
	created []entities.Notification
}

func (m *mockInboxRepoForBroadcast) List(context.Context, string, int, *inbox.Cursor) (inbox.Page, error) {
	return inbox.Page{}, nil
}
func (m *mockInboxRepoForBroadcast) UnreadCount(context.Context, string) (int64, error) {
	return 0, nil
}
func (m *mockInboxRepoForBroadcast) MarkRead(context.Context, string, string) (bool, error) {
	return true, nil
}
func (m *mockInboxRepoForBroadcast) Create(context.Context, *entities.Notification) error {
	return nil
}
func (m *mockInboxRepoForBroadcast) CreateBatch(_ context.Context, notifs []entities.Notification) error {
	m.created = append(m.created, notifs...)
	return nil
}

type mockDeviceRepoForBroadcast struct {
	tokens []string
}

func (m *mockDeviceRepoForBroadcast) RegisterDevice(context.Context, *entities.UserDevice) error {
	return nil
}
func (m *mockDeviceRepoForBroadcast) RevokeToken(context.Context, string, string) error {
	return nil
}
func (m *mockDeviceRepoForBroadcast) GetTokensByUser(context.Context, string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForBroadcast) GetTokensByUsers(context.Context, []string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForBroadcast) GetAllTokens(context.Context) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForBroadcast) CountActiveTokens(context.Context) (int64, error) {
	return int64(len(m.tokens)), nil
}

type mockPushSenderForBroadcast struct {
	sentCount int
}

func (m *mockPushSenderForBroadcast) Send(context.Context, string, string, string, map[string]string) error {
	m.sentCount++
	return nil
}

func TestBroadcastClassroom_Validation(t *testing.T) {
	handler := commands.NewBroadcastClassroomHandler(
		&mockClassroomRepoForBroadcast{},
		&mockInboxRepoForBroadcast{},
		&mockDeviceRepoForBroadcast{},
		&mockPushSenderForBroadcast{},
	)

	_, err := handler.Handle(context.Background(), commands.BroadcastClassroomCommand{
		ClassroomID: "",
		TeacherID:   "teacher-1",
		Title:       "Aviso",
		Message:     "Texto",
	})
	if err != commands.ErrEmptyClassroomID {
		t.Fatalf("expected ErrEmptyClassroomID, got %v", err)
	}

	_, err = handler.Handle(context.Background(), commands.BroadcastClassroomCommand{
		ClassroomID: "cls-1",
		TeacherID:   "teacher-1",
		Title:       "",
		Message:     "Texto",
	})
	if err != commands.ErrEmptyClassroomTitle {
		t.Fatalf("expected ErrEmptyClassroomTitle, got %v", err)
	}

	_, err = handler.Handle(context.Background(), commands.BroadcastClassroomCommand{
		ClassroomID: "cls-1",
		TeacherID:   "teacher-1",
		Title:       "Aviso",
		Message:     "",
	})
	if err != commands.ErrEmptyClassroomMessage {
		t.Fatalf("expected ErrEmptyClassroomMessage, got %v", err)
	}
}

func TestBroadcastClassroom_ForbiddenTeacher(t *testing.T) {
	clsRepo := &mockClassroomRepoForBroadcast{
		classroom: &entities.Classroom{
			ID:        "cls-1",
			TeacherID: "other-teacher",
		},
	}
	handler := commands.NewBroadcastClassroomHandler(
		clsRepo,
		&mockInboxRepoForBroadcast{},
		&mockDeviceRepoForBroadcast{},
		&mockPushSenderForBroadcast{},
	)

	_, err := handler.Handle(context.Background(), commands.BroadcastClassroomCommand{
		ClassroomID: "cls-1",
		TeacherID:   "teacher-attacker",
		Title:       "Aviso",
		Message:     "Mensaje",
	})
	if err != commands.ErrForbiddenTeacher {
		t.Fatalf("expected ErrForbiddenTeacher, got %v", err)
	}
}

func TestBroadcastClassroom_Success(t *testing.T) {
	clsRepo := &mockClassroomRepoForBroadcast{
		classroom: &entities.Classroom{
			ID:        "cls-1",
			TeacherID: "teacher-1",
		},
		enrollments: []entities.ClassroomEnrollment{
			{StudentID: "student-1"},
			{StudentID: "student-2"},
		},
	}
	inboxRepo := &mockInboxRepoForBroadcast{}
	deviceRepo := &mockDeviceRepoForBroadcast{
		tokens: []string{"fcm-1", "fcm-2"},
	}
	pushSender := &mockPushSenderForBroadcast{}

	handler := commands.NewBroadcastClassroomHandler(
		clsRepo,
		inboxRepo,
		deviceRepo,
		pushSender,
	)

	result, err := handler.Handle(context.Background(), commands.BroadcastClassroomCommand{
		ClassroomID:   "cls-1",
		TeacherID:     "teacher-1",
		Title:         "Tarea de Phrasal Verbs",
		Message:       "Completar lección 2 para mañana.",
		ActionType:    "open_lesson",
		ActionPayload: "lesson-99",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RecipientsCount != 2 {
		t.Errorf("expected 2 recipients, got %d", result.RecipientsCount)
	}
	if result.DevicesCount != 2 {
		t.Errorf("expected 2 devices dispatched, got %d", result.DevicesCount)
	}
	if len(inboxRepo.created) != 2 {
		t.Errorf("expected 2 notifications in inbox, got %d", len(inboxRepo.created))
	}
	if *inboxRepo.created[0].ActionURL != "/lessons/lesson-99" {
		t.Errorf("expected action url '/lessons/lesson-99', got %s", *inboxRepo.created[0].ActionURL)
	}

	resultCourse, err := handler.Handle(context.Background(), commands.BroadcastClassroomCommand{
		ClassroomID:   "cls-1",
		TeacherID:     "teacher-1",
		Title:         "Repaso del Curso",
		Message:       "Ver temario general.",
		ActionType:    "open_course",
		ActionPayload: "course-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultCourse.RecipientsCount != 2 {
		t.Errorf("expected 2 recipients, got %d", resultCourse.RecipientsCount)
	}
	if *inboxRepo.created[len(inboxRepo.created)-1].ActionURL != "/courses/course-123" {
		t.Errorf("expected action url '/courses/course-123', got %s", *inboxRepo.created[len(inboxRepo.created)-1].ActionURL)
	}
}
