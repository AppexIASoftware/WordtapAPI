package commands

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrSimulationDisabled   = errors.New("course purchase simulation is disabled")
	ErrInvalidPurchase      = errors.New("invalid course purchase request")
	ErrCourseNotPurchasable = errors.New("course is not available for purchase")
	ErrIdempotencyConflict  = errors.New("idempotency key belongs to another course")
)

const TeacherCourseCurrency = "PEN"

// SimulatedCoursePurchase is a test-only entitlement, never a payment receipt.
type SimulatedCoursePurchase struct {
	ID          string    `json:"id"`
	UserID      string    `json:"-"`
	CourseID    string    `json:"course_id"`
	AmountCents int       `json:"amount_cents"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
}

type SimulatedPurchaseRepository interface {
	Purchase(ctx context.Context, userID, courseID, key string) (*SimulatedCoursePurchase, error)
	Find(ctx context.Context, userID, courseID string) (*SimulatedCoursePurchase, error)
}

type SimulatePurchaseHandler struct {
	repo    SimulatedPurchaseRepository
	enabled bool
}

func NewSimulatePurchaseHandler(repo SimulatedPurchaseRepository, enabled bool) *SimulatePurchaseHandler {
	return &SimulatePurchaseHandler{repo: repo, enabled: enabled}
}

// Simulation requires an explicit flag and a known non-production environment.
func SimulationEnabled(environment, flag string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "local", "development", "dev", "test":
		return strings.TrimSpace(flag) == "true"
	default:
		return false
	}
}

func (h *SimulatePurchaseHandler) Purchase(ctx context.Context, userID, courseID, key string) (*SimulatedCoursePurchase, error) {
	if !h.enabled {
		return nil, ErrSimulationDisabled
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(courseID) == "" || strings.TrimSpace(key) != key || len(key) == 0 || len(key) > 128 {
		return nil, ErrInvalidPurchase
	}
	return h.repo.Purchase(ctx, userID, courseID, key)
}

func (h *SimulatePurchaseHandler) Find(ctx context.Context, userID, courseID string) (*SimulatedCoursePurchase, error) {
	if !h.enabled {
		return nil, ErrSimulationDisabled
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(courseID) == "" {
		return nil, ErrInvalidPurchase
	}
	return h.repo.Find(ctx, userID, courseID)
}
