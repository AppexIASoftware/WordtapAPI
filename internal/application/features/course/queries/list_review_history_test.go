package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

type historyStub struct {
	called bool
	err    error
}

func (s *historyStub) ListDecided(context.Context) ([]entities.CourseReviewRequest, error) {
	s.called = true
	return []entities.CourseReviewRequest{{ID: "approved", Status: entities.CourseReviewStatusApproved}}, s.err
}

func TestListReviewHistoryAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      string
		role    entities.UserRole
		allowed bool
	}{
		{"admin", "admin", entities.RoleAdmin, true},
		{"moderator", "mod", entities.RoleModerator, true},
		{"teacher", "teacher", entities.RoleInstructor, false},
		{"student", "student", entities.RoleStudent, false},
		{"missing identity", "", entities.RoleAdmin, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &historyStub{}
			items, err := NewListReviewHistoryHandler(repo).Handle(context.Background(), ListCourseReviewsQuery{RequesterID: tc.id, RequesterRole: tc.role})
			if tc.allowed != (err == nil) || repo.called != tc.allowed {
				t.Fatalf("allowed=%v called=%v err=%v", tc.allowed, repo.called, err)
			}
			if tc.allowed && (len(items) != 1 || items[0].ID != "approved") {
				t.Fatalf("unexpected history: %v", items)
			}
		})
	}
}

func TestListReviewHistoryPropagatesRepositoryError(t *testing.T) {
	want := errors.New("database unavailable")
	_, err := NewListReviewHistoryHandler(&historyStub{err: want}).Handle(context.Background(), ListCourseReviewsQuery{RequesterID: "admin", RequesterRole: entities.RoleAdmin})
	if !errors.Is(err, want) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
