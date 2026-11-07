package commands

import (
	"context"
	"errors"
	"testing"
)

type purchaseStub struct{ calls int }

func (s *purchaseStub) Purchase(_ context.Context, user, course, key string) (*SimulatedCoursePurchase, error) {
	s.calls++
	return &SimulatedCoursePurchase{UserID: user, CourseID: course, AmountCents: 2750, Currency: TeacherCourseCurrency}, nil
}
func (s *purchaseStub) Find(_ context.Context, user, course string) (*SimulatedCoursePurchase, error) {
	s.calls++
	return &SimulatedCoursePurchase{UserID: user, CourseID: course}, nil
}

func TestSimulationEnabled(t *testing.T) {
	for _, tc := range []struct {
		env, flag string
		want      bool
	}{
		{"", "true", false}, {"production", "true", false}, {"staging", "true", false},
		{"local", "", false}, {"local", "false", false}, {"local", "TRUE", false},
		{"local", "true", true}, {"development", "true", true}, {"dev", "true", true}, {"test", "true", true},
	} {
		t.Run(tc.env+"/"+tc.flag, func(t *testing.T) {
			if got := SimulationEnabled(tc.env, tc.flag); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSimulatePurchaseValidatesBeforeRepository(t *testing.T) {
	for _, tc := range []struct {
		name              string
		enabled           bool
		user, course, key string
		want              error
	}{
		{"disabled", false, "u", "c", "k", ErrSimulationDisabled},
		{"missing user", true, "", "c", "k", ErrInvalidPurchase},
		{"missing course", true, "u", "", "k", ErrInvalidPurchase},
		{"missing key", true, "u", "c", "", ErrInvalidPurchase},
		{"untrimmed key", true, "u", "c", " key ", ErrInvalidPurchase},
		{"success", true, "u", "c", "k", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &purchaseStub{}
			got, err := NewSimulatePurchaseHandler(repo, tc.enabled).Purchase(context.Background(), tc.user, tc.course, tc.key)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error %v, want %v", err, tc.want)
			}
			if tc.want != nil && repo.calls != 0 {
				t.Fatal("invalid request reached repository")
			}
			if tc.want == nil && (got.AmountCents != 2750 || got.Currency != "PEN") {
				t.Fatalf("unexpected result: %+v", got)
			}
		})
	}
}

func TestDisabledSimulationCannotReadTestRights(t *testing.T) {
	repo := &purchaseStub{}
	_, err := NewSimulatePurchaseHandler(repo, false).Find(context.Background(), "u", "c")
	if !errors.Is(err, ErrSimulationDisabled) || repo.calls != 0 {
		t.Fatalf("disabled lookup: %v, calls %d", err, repo.calls)
	}
}
