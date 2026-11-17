package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/inbox"
	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/vault/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
)

type mockBankRepoForDispatch struct {
	item *entities.ContentBankItem
	err  error
}

func (m *mockBankRepoForDispatch) ListTypes(context.Context) ([]entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockBankRepoForDispatch) FindTypeBySlug(context.Context, string) (*entities.ContentBankType, error) {
	return nil, nil
}
func (m *mockBankRepoForDispatch) ListCategories(context.Context) ([]entities.ContentCategory, error) {
	return nil, nil
}
func (m *mockBankRepoForDispatch) CreateCategory(context.Context, *entities.ContentCategory) error {
	return nil
}
func (m *mockBankRepoForDispatch) ListItems(context.Context, repositories.ContentBankFilter) ([]entities.ContentBankItem, int64, error) {
	return nil, 0, nil
}
func (m *mockBankRepoForDispatch) FindItemByID(context.Context, string) (*entities.ContentBankItem, error) {
	return m.item, m.err
}
func (m *mockBankRepoForDispatch) CreateItem(context.Context, *entities.ContentBankItem) error {
	return nil
}
func (m *mockBankRepoForDispatch) UpdateItem(context.Context, *entities.ContentBankItem) error {
	return nil
}
func (m *mockBankRepoForDispatch) DeleteItem(context.Context, string) error {
	return nil
}

type mockUserRepoForDispatch struct {
	userIDs []string
	err     error
}

func (m *mockUserRepoForDispatch) FindByID(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepoForDispatch) FindByEmail(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepoForDispatch) FindAuthAccount(context.Context, string, string) (*entities.UserAuthAccount, error) {
	return nil, nil
}
func (m *mockUserRepoForDispatch) CreateWithInitialState(context.Context, *entities.User, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepoForDispatch) CreateAuthAccount(context.Context, *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepoForDispatch) CreateSession(context.Context, *entities.UserSession) error {
	return nil
}
func (m *mockUserRepoForDispatch) FindSessionByTokenHash(context.Context, string) (*entities.UserSession, error) {
	return nil, nil
}
func (m *mockUserRepoForDispatch) RotateSession(context.Context, string, *entities.UserSession) (bool, error) {
	return false, nil
}
func (m *mockUserRepoForDispatch) RevokeSession(context.Context, string) error {
	return nil
}
func (m *mockUserRepoForDispatch) RevokeAllSessions(context.Context, string) error {
	return nil
}
func (m *mockUserRepoForDispatch) UpdateUser(context.Context, *entities.User) error {
	return nil
}
func (m *mockUserRepoForDispatch) ListUserIDs(context.Context, *entities.UserRole, *string) ([]string, error) {
	return m.userIDs, m.err
}

type mockClassroomRepoForDispatch struct {
	classroom   *entities.Classroom
	enrollments []entities.ClassroomEnrollment
}

func (m *mockClassroomRepoForDispatch) Create(context.Context, *entities.Classroom) error { return nil }
func (m *mockClassroomRepoForDispatch) FindByID(context.Context, string) (*entities.Classroom, error) {
	return m.classroom, nil
}
func (m *mockClassroomRepoForDispatch) FindByCode(context.Context, string) (*entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForDispatch) FindByTeacher(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForDispatch) FindByStudent(context.Context, string) ([]entities.Classroom, error) {
	return nil, nil
}
func (m *mockClassroomRepoForDispatch) EnrollStudent(context.Context, *entities.ClassroomEnrollment) error {
	return nil
}
func (m *mockClassroomRepoForDispatch) FindEnrollment(context.Context, string, string) (*entities.ClassroomEnrollment, error) {
	return nil, nil
}
func (m *mockClassroomRepoForDispatch) ListEnrollments(context.Context, string) ([]entities.ClassroomEnrollment, error) {
	return m.enrollments, nil
}
func (m *mockClassroomRepoForDispatch) ListStumbles(context.Context, string) ([]entities.StudentStumble, error) {
	return nil, nil
}
func (m *mockClassroomRepoForDispatch) CreateStumble(context.Context, *entities.StudentStumble) error {
	return nil
}
func (m *mockClassroomRepoForDispatch) Update(context.Context, *entities.Classroom) error { return nil }

type mockInboxRepoForDispatch struct {
	created []entities.Notification
}

func (m *mockInboxRepoForDispatch) List(context.Context, string, int, *inbox.Cursor) (inbox.Page, error) {
	return inbox.Page{}, nil
}
func (m *mockInboxRepoForDispatch) UnreadCount(context.Context, string) (int64, error) { return 0, nil }
func (m *mockInboxRepoForDispatch) MarkRead(context.Context, string, string) (bool, error) {
	return true, nil
}
func (m *mockInboxRepoForDispatch) Create(context.Context, *entities.Notification) error { return nil }
func (m *mockInboxRepoForDispatch) CreateBatch(_ context.Context, notifs []entities.Notification) error {
	m.created = append(m.created, notifs...)
	return nil
}

type mockDeviceRepoForDispatch struct {
	tokens []string
}

func (m *mockDeviceRepoForDispatch) RegisterDevice(context.Context, *entities.UserDevice) error {
	return nil
}
func (m *mockDeviceRepoForDispatch) RevokeToken(context.Context, string, string) error { return nil }
func (m *mockDeviceRepoForDispatch) GetTokensByUser(context.Context, string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForDispatch) GetTokensByUsers(context.Context, []string) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForDispatch) GetAllTokens(context.Context) ([]string, error) {
	return m.tokens, nil
}
func (m *mockDeviceRepoForDispatch) CountActiveTokens(context.Context) (int64, error) {
	return int64(len(m.tokens)), nil
}

type mockPushSenderForDispatch struct {
	sentCount int
}

func (m *mockPushSenderForDispatch) Send(context.Context, string, string, string, map[string]string) error {
	m.sentCount++
	return nil
}

func TestDispatchVaultWord_Validation(t *testing.T) {
	handler := commands.NewDispatchVaultWordHandler(
		&mockBankRepoForDispatch{},
		&mockClassroomRepoForDispatch{},
		&mockUserRepoForDispatch{},
		&mockInboxRepoForDispatch{},
		&mockDeviceRepoForDispatch{},
		&mockPushSenderForDispatch{},
	)

	_, err := handler.Handle(context.Background(), commands.DispatchVaultWordCommand{
		ItemVaultID: "",
	})
	if err != commands.ErrEmptyVaultItemID {
		t.Fatalf("expected ErrEmptyVaultItemID, got %v", err)
	}

	_, err = handler.Handle(context.Background(), commands.DispatchVaultWordCommand{
		ItemVaultID: "missing-item",
	})
	if err != commands.ErrVaultItemNotFound {
		t.Fatalf("expected ErrVaultItemNotFound, got %v", err)
	}
}

func TestDispatchVaultWord_Success_CEFR(t *testing.T) {
	ipa := "/əˈkwajɚ/"
	bankItem := &entities.ContentBankItem{
		ID:         "vault-item-1",
		PromptText: "adquirir",
		TargetText: "acquire",
		IPA:        &ipa,
		CEFRLevel:  "B2",
	}

	bankRepo := &mockBankRepoForDispatch{item: bankItem}
	clsRepo := &mockClassroomRepoForDispatch{}
	userRepo := &mockUserRepoForDispatch{userIDs: []string{"student-1", "student-2"}}
	inboxRepo := &mockInboxRepoForDispatch{}
	deviceRepo := &mockDeviceRepoForDispatch{tokens: []string{"fcm-1", "fcm-2"}}
	pushSender := &mockPushSenderForDispatch{}

	handler := commands.NewDispatchVaultWordHandler(
		bankRepo,
		clsRepo,
		userRepo,
		inboxRepo,
		deviceRepo,
		pushSender,
	)

	cefr := "B2"
	result, err := handler.Handle(context.Background(), commands.DispatchVaultWordCommand{
		ItemVaultID:     "vault-item-1",
		TargetAudience:  "cefr",
		TargetCEFRLevel: &cefr,
		SenderUserID:    "admin-1",
		SenderRole:      entities.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Word != "acquire" {
		t.Errorf("expected word 'acquire', got %s", result.Word)
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
}

func TestDispatchVaultWord_RBACAndOwnership(t *testing.T) {
	ownerID := "teacher-owner"
	privateItem := &entities.ContentBankItem{
		ID:         "vault-private-1",
		PromptText: "lograr",
		TargetText: "achieve",
		CEFRLevel:  "B1",
		CreatedBy:  &ownerID,
	}
	officialItem := &entities.ContentBankItem{
		ID:         "vault-official-1",
		PromptText: "casa",
		TargetText: "house",
		CEFRLevel:  "A1",
	}

	t.Run("instructor cannot dispatch another teacher private vault item", func(t *testing.T) {
		clsID := "cls-1"
		handler := commands.NewDispatchVaultWordHandler(
			&mockBankRepoForDispatch{item: privateItem},
			&mockClassroomRepoForDispatch{classroom: &entities.Classroom{ID: clsID, TeacherID: "teacher-other"}},
			&mockUserRepoForDispatch{},
			&mockInboxRepoForDispatch{},
			&mockDeviceRepoForDispatch{},
			&mockPushSenderForDispatch{},
		)

		_, err := handler.Handle(context.Background(), commands.DispatchVaultWordCommand{
			ItemVaultID:       "vault-private-1",
			TargetAudience:    "classroom",
			TargetClassroomID: &clsID,
			SenderUserID:      "teacher-other",
			SenderRole:        entities.RoleInstructor,
		})
		if !errors.Is(err, commands.ErrVaultItemNotFound) {
			t.Fatalf("expected ErrVaultItemNotFound, got %v", err)
		}
	})

	t.Run("instructor blocked from cefr and all audiences", func(t *testing.T) {
		handler := commands.NewDispatchVaultWordHandler(
			&mockBankRepoForDispatch{item: officialItem},
			&mockClassroomRepoForDispatch{},
			&mockUserRepoForDispatch{userIDs: []string{"student-1"}},
			&mockInboxRepoForDispatch{},
			&mockDeviceRepoForDispatch{},
			&mockPushSenderForDispatch{},
		)

		for _, aud := range []string{"cefr", "all"} {
			_, err := handler.Handle(context.Background(), commands.DispatchVaultWordCommand{
				ItemVaultID:    "vault-official-1",
				TargetAudience: aud,
				SenderUserID:   "teacher-owner",
				SenderRole:     entities.RoleInstructor,
			})
			if !errors.Is(err, commands.ErrGlobalDispatchForbidden) {
				t.Fatalf("audience %s: expected ErrGlobalDispatchForbidden, got %v", aud, err)
			}
		}
	})

	t.Run("fails closed on missing sender role and rejects unknown audience", func(t *testing.T) {
		handler := commands.NewDispatchVaultWordHandler(
			&mockBankRepoForDispatch{item: officialItem},
			&mockClassroomRepoForDispatch{},
			&mockUserRepoForDispatch{},
			&mockInboxRepoForDispatch{},
			&mockDeviceRepoForDispatch{},
			&mockPushSenderForDispatch{},
		)

		_, err := handler.Handle(context.Background(), commands.DispatchVaultWordCommand{
			ItemVaultID:    "vault-official-1",
			TargetAudience: "all",
			SenderUserID:   "user-1",
			SenderRole:     "",
		})
		if !errors.Is(err, commands.ErrClassroomForbidden) {
			t.Fatalf("expected ErrClassroomForbidden for empty role, got %v", err)
		}

		_, err = handler.Handle(context.Background(), commands.DispatchVaultWordCommand{
			ItemVaultID:    "vault-official-1",
			TargetAudience: "unknown",
			SenderUserID:   "admin-1",
			SenderRole:     entities.RoleAdmin,
		})
		if !errors.Is(err, commands.ErrInvalidTargetAudience) {
			t.Fatalf("expected ErrInvalidTargetAudience, got %v", err)
		}
	})
}
