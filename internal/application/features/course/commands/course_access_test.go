package commands

import (
	"context"
	"errors"
	"testing"
)

type courseAccessStub struct{ calls int }

func (s *courseAccessStub) GrantTestAccess(_ context.Context, userID, courseID string) error {
	s.calls++
	return nil
}

func TestGrantTestCourseAccess(t *testing.T) {
	for _, tc := range []struct {
		name, user, course string
		enabled            bool
		want               error
	}{
		{"disabled", "u", "c", false, ErrTestAccessDisabled},
		{"missing user", "", "c", true, ErrInvalidCourseAccess},
		{"missing course", "u", "", true, ErrInvalidCourseAccess},
		{"valid", "u", "c", true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &courseAccessStub{}
			err := NewGrantTestCourseAccessHandler(repo, tc.enabled).Handle(context.Background(), tc.user, tc.course)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if (repo.calls == 1) != (tc.want == nil) {
				t.Fatalf("repository calls: %d", repo.calls)
			}
		})
	}
}
