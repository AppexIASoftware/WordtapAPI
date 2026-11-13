package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type staffRepoStub struct {
	usersByEmail map[string]*entities.User
	usersByID    map[string]*entities.User
	revokedIDs   []string
}

func newStaffRepoStub() *staffRepoStub {
	return &staffRepoStub{
		usersByEmail: map[string]*entities.User{},
		usersByID:    map[string]*entities.User{},
	}
}

func (s *staffRepoStub) FindByID(_ context.Context, id string) (*entities.User, error) {
	if u, ok := s.usersByID[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *staffRepoStub) FindByEmail(_ context.Context, email string) (*entities.User, error) {
	if u, ok := s.usersByEmail[email]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *staffRepoStub) FindAuthAccount(context.Context, string, string) (*entities.UserAuthAccount, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *staffRepoStub) CreateWithInitialState(_ context.Context, user *entities.User, _ *entities.UserAuthAccount) error {
	if user.ID == "" {
		user.ID = "user-generated-1"
	}
	user.CreatedAt = time.Now()
	s.usersByEmail[user.Email] = user
	s.usersByID[user.ID] = user
	return nil
}

func (s *staffRepoStub) CreateAuthAccount(context.Context, *entities.UserAuthAccount) error { return nil }
func (s *staffRepoStub) CreateSession(context.Context, *entities.UserSession) error         { return nil }
func (s *staffRepoStub) FindSessionByTokenHash(context.Context, string) (*entities.UserSession, error) {
	return nil, gorm.ErrRecordNotFound
}
func (s *staffRepoStub) RotateSession(context.Context, string, *entities.UserSession) (bool, error) {
	return false, nil
}
func (s *staffRepoStub) RevokeSession(context.Context, string) error { return nil }
func (s *staffRepoStub) RevokeAllSessions(_ context.Context, userID string) error {
	s.revokedIDs = append(s.revokedIDs, userID)
	return nil
}
func (s *staffRepoStub) UpdateUser(_ context.Context, user *entities.User) error {
	s.usersByEmail[user.Email] = user
	s.usersByID[user.ID] = user
	return nil
}
func (s *staffRepoStub) ListUserIDs(context.Context, *entities.UserRole, *string) ([]string, error) {
	return nil, nil
}
func (s *staffRepoStub) ListStaff(context.Context) ([]entities.User, error) {
	return nil, nil
}
func (s *staffRepoStub) DeleteUser(_ context.Context, userID string) error {
	if u, ok := s.usersByID[userID]; ok {
		delete(s.usersByEmail, u.Email)
		delete(s.usersByID, userID)
		return nil
	}
	return gorm.ErrRecordNotFound
}

func TestProvisionStaffCreatesNewAdminWithVerifiedEmail(t *testing.T) {
	repo := newStaffRepoStub()
	h := NewProvisionStaffHandler(repo)

	res, err := h.Handle(context.Background(), ProvisionStaffCommand{
		RequesterID: "admin-1",
		Email:       "  New.Admin@Wordtap.io ",
		Name:        "Ana Gómez",
		Role:        entities.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !res.Created {
		t.Fatalf("expected Created=true")
	}
	if res.User.Email != "new.admin@wordtap.io" {
		t.Fatalf("expected normalized email, got %q", res.User.Email)
	}
	if res.User.Role != entities.RoleAdmin || res.User.AccessTier != entities.AccessTierAdmin {
		t.Fatalf("expected admin role and tier, got %s / %s", res.User.Role, res.User.AccessTier)
	}
	stored := repo.usersByEmail["new.admin@wordtap.io"]
	if stored == nil || stored.EmailVerifiedAt == nil || !stored.IsActive {
		t.Fatalf("expected stored user to be active and email-verified for Google OAuth linking")
	}
}

func TestProvisionStaffPromotesExistingStudentAndRevokesSessions(t *testing.T) {
	repo := newStaffRepoStub()
	repo.usersByEmail["user@wordtap.io"] = &entities.User{
		ID:         "u-2",
		Email:      "user@wordtap.io",
		Name:       "user",
		Role:       entities.RoleStudent,
		AccessTier: entities.AccessTierFree,
		IsActive:   false,
	}
	repo.usersByID["u-2"] = repo.usersByEmail["user@wordtap.io"]

	h := NewProvisionStaffHandler(repo)
	res, err := h.Handle(context.Background(), ProvisionStaffCommand{
		RequesterID: "admin-1",
		Email:       "user@wordtap.io",
		Name:        "Carlos Ruiz",
		Role:        entities.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Created {
		t.Fatalf("expected Created=false on promotion")
	}
	if res.User.Role != entities.RoleAdmin || !res.User.IsActive || res.User.Name != "Carlos Ruiz" {
		t.Fatalf("unexpected promoted user: %+v", res.User)
	}
	if len(repo.revokedIDs) != 1 || repo.revokedIDs[0] != "u-2" {
		t.Fatalf("expected sessions revoked on role change, got %v", repo.revokedIDs)
	}
}

func TestProvisionStaffRejectsInvalidRoleOrEmail(t *testing.T) {
	h := NewProvisionStaffHandler(newStaffRepoStub())
	if _, err := h.Handle(context.Background(), ProvisionStaffCommand{Email: "bad-email", Role: entities.RoleAdmin}); !errors.Is(err, ErrInvalidStaffEmail) {
		t.Fatalf("expected ErrInvalidStaffEmail, got %v", err)
	}
	if _, err := h.Handle(context.Background(), ProvisionStaffCommand{Email: "a@b.com", Role: entities.RoleStudent}); !errors.Is(err, ErrInvalidStaffRole) {
		t.Fatalf("expected ErrInvalidStaffRole, got %v", err)
	}
}

func TestUpdateStaffStatusPreventsSelfSuspensionAndRevokesOnSuspend(t *testing.T) {
	repo := newStaffRepoStub()
	repo.usersByID["admin-1"] = &entities.User{ID: "admin-1", Email: "a@b.com", Role: entities.RoleAdmin, IsActive: true}
	repo.usersByID["admin-2"] = &entities.User{ID: "admin-2", Email: "c@d.com", Role: entities.RoleAdmin, IsActive: true}

	h := NewUpdateStaffStatusHandler(repo)

	if _, err := h.Handle(context.Background(), UpdateStaffStatusCommand{
		RequesterID:  "admin-1",
		TargetUserID: "admin-1",
		IsActive:     false,
	}); !errors.Is(err, ErrCannotSuspendSelf) {
		t.Fatalf("expected ErrCannotSuspendSelf, got %v", err)
	}

	dto, err := h.Handle(context.Background(), UpdateStaffStatusCommand{
		RequesterID:  "admin-1",
		TargetUserID: "admin-2",
		IsActive:     false,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dto.IsActive {
		t.Fatalf("expected user suspended")
	}
	if len(repo.revokedIDs) != 1 || repo.revokedIDs[0] != "admin-2" {
		t.Fatalf("expected sessions revoked on suspension, got %v", repo.revokedIDs)
	}
}

func TestDeleteStaffPreventsSelfDeleteAndRemovesTarget(t *testing.T) {
	repo := newStaffRepoStub()
	repo.usersByID["admin-1"] = &entities.User{ID: "admin-1", Email: "a@b.com", Role: entities.RoleAdmin, IsActive: true}
	repo.usersByID["admin-2"] = &entities.User{ID: "admin-2", Email: "c@d.com", Role: entities.RoleAdmin, IsActive: true}
	repo.usersByEmail["c@d.com"] = repo.usersByID["admin-2"]

	h := NewDeleteStaffHandler(repo)

	if err := h.Handle(context.Background(), DeleteStaffCommand{
		RequesterID:  "admin-1",
		TargetUserID: "admin-1",
	}); !errors.Is(err, ErrCannotDeleteSelf) {
		t.Fatalf("expected ErrCannotDeleteSelf, got %v", err)
	}

	if err := h.Handle(context.Background(), DeleteStaffCommand{
		RequesterID:  "admin-1",
		TargetUserID: "admin-2",
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := repo.usersByID["admin-2"]; ok {
		t.Fatalf("expected admin-2 to be deleted")
	}
}
