package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/teacher_application/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	"gorm.io/gorm"
)

type mockUserRepo struct {
	user *entities.User
	err  error
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*entities.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepo) CreateWithInitialState(ctx context.Context, u *entities.User, a *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepo) UpdateUser(ctx context.Context, u *entities.User) error { return nil }
func (m *mockUserRepo) FindAuthAccount(ctx context.Context, p, a string) (*entities.UserAuthAccount, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepo) CreateAuthAccount(ctx context.Context, a *entities.UserAuthAccount) error {
	return nil
}
func (m *mockUserRepo) CreateSession(ctx context.Context, s *entities.UserSession) error { return nil }
func (m *mockUserRepo) FindSessionByTokenHash(ctx context.Context, t string) (*entities.UserSession, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockUserRepo) RevokeSession(ctx context.Context, t string) error { return nil }
func (m *mockUserRepo) RevokeAllSessions(ctx context.Context, u string) error { return nil }

type mockAppRepo struct {
	created  *entities.TeacherApplication
	existing *entities.TeacherApplication
	appErr   error
}

func (m *mockAppRepo) Create(ctx context.Context, app *entities.TeacherApplication) error {
	m.created = app
	return nil
}
func (m *mockAppRepo) FindByID(ctx context.Context, id string) (*entities.TeacherApplication, error) {
	return m.existing, m.appErr
}
func (m *mockAppRepo) FindByUserID(ctx context.Context, userID string) (*entities.TeacherApplication, error) {
	if m.existing != nil {
		return m.existing, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockAppRepo) List(ctx context.Context, s *entities.ApplicationStatus, l, o int) ([]entities.TeacherApplication, int64, error) {
	return nil, 0, nil
}
func (m *mockAppRepo) Update(ctx context.Context, app *entities.TeacherApplication) error { return nil }
func (m *mockAppRepo) ApproveInTx(ctx context.Context, appID string, revID string) error {
	if m.existing != nil {
		m.existing.Status = entities.ApplicationStatusApproved
	}
	return nil
}
func (m *mockAppRepo) RejectInTx(ctx context.Context, appID string, revID string, reason string) error {
	if m.existing != nil {
		m.existing.Status = entities.ApplicationStatusRejected
	}
	return nil
}
func (m *mockAppRepo) SuspendInTx(ctx context.Context, appID string, revID string, reason string) error {
	if m.existing != nil {
		m.existing.Status = entities.ApplicationStatusSuspended
	}
	return nil
}
func (m *mockAppRepo) ReactivateInTx(ctx context.Context, appID string, revID string) error {
	if m.existing != nil {
		m.existing.Status = entities.ApplicationStatusApproved
	}
	return nil
}
func (m *mockAppRepo) DeleteInTx(ctx context.Context, appID string) error {
	m.existing = nil
	return nil
}

func TestApplyTeacherHandler(t *testing.T) {
	student := &entities.User{ID: "usr-1", Role: entities.RoleStudent}
	uRepo := &mockUserRepo{user: student}
	aRepo := &mockAppRepo{}

	handler := commands.NewApplyTeacherHandler(aRepo, uRepo)
	app, err := handler.Handle(context.Background(), commands.ApplyTeacherCommand{
		UserID: "usr-1",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if app.Status != entities.ApplicationStatusPending {
		t.Fatalf("expected pending status, got %s", app.Status)
	}
}

func TestReviewApplicationHandler(t *testing.T) {
	app := &entities.TeacherApplication{
		ID:        "app-1",
		UserID:    "usr-1",
		Status:    entities.ApplicationStatusPending,
		CreatedAt: time.Now(),
	}
	aRepo := &mockAppRepo{existing: app}

	handler := commands.NewReviewApplicationHandler(aRepo)
	reviewed, err := handler.Handle(context.Background(), commands.ReviewApplicationCommand{
		ApplicationID: "app-1",
		ReviewerID:    "admin-1",
		ReviewerRole:  entities.RoleAdmin,
		Action:        "approve",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reviewed.Status != entities.ApplicationStatusApproved {
		t.Fatalf("expected approved status, got %s", reviewed.Status)
	}
}
