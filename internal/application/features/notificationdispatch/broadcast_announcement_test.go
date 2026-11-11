package notificationdispatch_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationdispatch"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type mockUserRepoForAnnouncement struct {
	userIDs []string
	err     error
}

func (m *mockUserRepoForAnnouncement) FindByID(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepoForAnnouncement) FindByEmail(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepoForAnnouncement) FindAuthAccount(context.Context, string, string) (*entities.UserAuthAccount, error) {
	return nil, nil
}
func (m *mockUserRepoForAnnouncement) CreateWithInitialState(context.Context, *entities.User, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepoForAnnouncement) CreateAuthAccount(context.Context, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepoForAnnouncement) CreateSession(context.Context, *entities.UserSession) error {
	return nil
}
func (m *mockUserRepoForAnnouncement) FindSessionByTokenHash(context.Context, string) (*entities.UserSession, error) {
	return nil, nil
}
func (m *mockUserRepoForAnnouncement) RotateSession(context.Context, string, *entities.UserSession) (bool, error) {
	return false, nil
}
func (m *mockUserRepoForAnnouncement) RevokeSession(context.Context, string) error {
	return nil
}
func (m *mockUserRepoForAnnouncement) RevokeAllSessions(context.Context, string) error {
	return nil
}
func (m *mockUserRepoForAnnouncement) UpdateUser(context.Context, *entities.User) error {
	return nil
}
func (m *mockUserRepoForAnnouncement) ListUserIDs(context.Context, *entities.UserRole, *string) ([]string, error) {
	return m.userIDs, m.err
}

func TestBroadcastAnnouncement_Validation(t *testing.T) {
	handler := notificationdispatch.NewBroadcastAnnouncementHandler(
		&mockUserRepoForAnnouncement{},
		&mockInboxRepo{},
		&mockDeviceRepo{},
		&mockPushSender{},
	)

	_, err := handler.Handle(context.Background(), notificationdispatch.BroadcastAnnouncementCommand{
		Title: "",
		Body:  "Mensaje de prueba",
	})
	if err != notificationdispatch.ErrEmptyAnnouncementTitle {
		t.Fatalf("expected ErrEmptyAnnouncementTitle, got %v", err)
	}

	_, err = handler.Handle(context.Background(), notificationdispatch.BroadcastAnnouncementCommand{
		Title: "Título",
		Body:  "",
	})
	if err != notificationdispatch.ErrEmptyAnnouncementBody {
		t.Fatalf("expected ErrEmptyAnnouncementBody, got %v", err)
	}
}

func TestBroadcastAnnouncement_Success(t *testing.T) {
	userRepo := &mockUserRepoForAnnouncement{
		userIDs: []string{"user-1", "user-2"},
	}
	inboxRepo := &mockInboxRepo{}
	deviceRepo := &mockDeviceRepo{
		tokens: []string{"token-1", "token-2"},
	}
	pushSender := &mockPushSender{}

	handler := notificationdispatch.NewBroadcastAnnouncementHandler(
		userRepo,
		inboxRepo,
		deviceRepo,
		pushSender,
	)

	url := "https://wordtap.app/catalog"
	result, err := handler.Handle(context.Background(), notificationdispatch.BroadcastAnnouncementCommand{
		Title:     "Nuevo Curso Disponible",
		Body:      "Explora el curso de Business English.",
		ActionURL: &url,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RecipientsCount != 2 {
		t.Errorf("expected 2 recipients, got %d", result.RecipientsCount)
	}
	if result.DevicesCount != 2 {
		t.Errorf("expected 2 devices, got %d", result.DevicesCount)
	}
	if pushSender.sentCount != 2 {
		t.Errorf("expected 2 pushes sent, got %d", pushSender.sentCount)
	}
}
