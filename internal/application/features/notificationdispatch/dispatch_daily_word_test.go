package notificationdispatch_test

import (
	"context"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/notificationdispatch"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type mockBankRepo struct {
	items []entities.ContentBankItem
	total int64
}

func (m *mockBankRepo) ListTypes(ctx context.Context) ([]entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockBankRepo) FindTypeBySlug(ctx context.Context, slug string) (*entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockBankRepo) ListCategories(ctx context.Context) ([]entities.ContentCategory, error) {
	return nil, nil
}
func (m *mockBankRepo) CreateCategory(ctx context.Context, category *entities.ContentCategory) error {
	return nil
}
func (m *mockBankRepo) ListItems(ctx context.Context, filter repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	return m.items, m.total, nil
}
func (m *mockBankRepo) FindItemByID(ctx context.Context, id string) (*entities.ContentBankItem, error) {
	return nil, nil
}
func (m *mockBankRepo) CreateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return nil
}
func (m *mockBankRepo) UpdateItem(ctx context.Context, item *entities.ContentBankItem) error {
	return nil
}
func (m *mockBankRepo) DeleteItem(ctx context.Context, id string) error {
	return nil
}

type mockInboxRepo struct {
	createdNotification *entities.Notification
}

func (m *mockInboxRepo) List(ctx context.Context, userID string, limit int, cursor *inbox.Cursor) (inbox.Page, error) {
	return inbox.Page{}, nil
}
func (m *mockInboxRepo) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}
func (m *mockInboxRepo) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	return true, nil
}
func (m *mockInboxRepo) Create(ctx context.Context, n *entities.Notification) error {
	m.createdNotification = n
	return nil
}
func (m *mockInboxRepo) CreateBatch(ctx context.Context, notifs []entities.Notification) error {
	if len(notifs) > 0 {
		m.createdNotification = &notifs[0]
	}
	return nil
}

type mockDeviceRepo struct {
	tokens []string
}

func (m *mockDeviceRepo) RegisterDevice(ctx context.Context, d *entities.UserDevice) error {
	return nil
}
func (m *mockDeviceRepo) RevokeToken(ctx context.Context, userID, token string) error {
	return nil
}
func (m *mockDeviceRepo) GetTokensByUser(ctx context.Context, userID string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepo) GetTokensByUsers(ctx context.Context, userIDs []string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepo) GetAllTokens(ctx context.Context) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepo) CountActiveTokens(ctx context.Context) (int64, error) {
	return int64(len(m.tokens)), nil
}

type mockPushSender struct {
	sentCount int
	lastTitle string
	lastBody  string
}

func (m *mockPushSender) Send(ctx context.Context, token, title, body string, data map[string]string) error {
	m.sentCount++
	m.lastTitle = title
	m.lastBody = body
	return nil
}

func TestDispatchDailyWord_Success(t *testing.T) {
	ipa := "/kæt/"
	bank := &mockBankRepo{
		items: []entities.ContentBankItem{
			{
				ID:         "v-1",
				PromptText: "gato",
				TargetText: "cat",
				IPA:        &ipa,
			},
		},
		total: 1,
	}
	inboxRepo := &mockInboxRepo{}
	deviceRepo := &mockDeviceRepo{tokens: []string{"fcm-device-token"}}
	pushSender := &mockPushSender{}

	handler := notificationdispatch.NewDispatchDailyWordHandler(bank, inboxRepo, deviceRepo, pushSender)

	res, err := handler.Handle(context.Background(), notificationdispatch.DispatchDailyWordCommand{
		UserID: "user-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Dispatched || res.Word != "cat" || res.DevicesCount != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}

	if inboxRepo.createdNotification == nil {
		t.Fatal("expected notification to be created in inbox")
	}
	if inboxRepo.createdNotification.UserID != "user-123" {
		t.Errorf("expected user-123, got %s", inboxRepo.createdNotification.UserID)
	}
	if pushSender.sentCount != 1 || pushSender.lastTitle != "Palabra del día: cat" {
		t.Errorf("unexpected push sender call: %+v", pushSender)
	}
}
