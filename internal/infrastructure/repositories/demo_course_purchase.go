package repositories

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/AppexIASoftware/WordtapAPI/internal/application/features/course/commands"
)

// DemoPurchaseRepository grants test-only course access from process memory.
//
// It exists so non-production environments can exercise the real lesson and
// game content without writing simulated payments to the database. Nothing is
// persisted: restarting the API clears every granted access.
type DemoPurchaseRepository struct {
	mu        sync.Mutex
	purchases map[string]commands.SimulatedCoursePurchase
	keys      map[string]string
}

func NewDemoPurchaseRepository() *DemoPurchaseRepository {
	return &DemoPurchaseRepository{
		purchases: make(map[string]commands.SimulatedCoursePurchase),
		keys:      make(map[string]string),
	}
}

func demoPurchaseKey(userID, courseID string) string {
	return userID + ":" + courseID
}

// Purchase records a demo access for one student and course. The receipt is
// reported with amount zero because no payment is being represented.
func (r *DemoPurchaseRepository) Purchase(_ context.Context, userID, courseID, key string) (*commands.SimulatedCoursePurchase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	keyID := userID + "\x00" + key
	if existingCourse, ok := r.keys[keyID]; ok && existingCourse != courseID {
		return nil, commands.ErrIdempotencyConflict
	}
	r.keys[keyID] = courseID

	accessID := demoPurchaseKey(userID, courseID)
	if purchase, ok := r.purchases[accessID]; ok {
		result := purchase
		return &result, nil
	}

	purchase := commands.SimulatedCoursePurchase{
		ID:          uuid.NewString(),
		UserID:      userID,
		CourseID:    courseID,
		AmountCents: 0,
		Currency:    commands.TeacherCourseCurrency,
		CreatedAt:   time.Now().UTC(),
	}
	r.purchases[accessID] = purchase
	result := purchase
	return &result, nil
}

func (r *DemoPurchaseRepository) Find(_ context.Context, userID, courseID string) (*commands.SimulatedCoursePurchase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	purchase, ok := r.purchases[demoPurchaseKey(userID, courseID)]
	if !ok {
		return nil, nil
	}
	result := purchase
	return &result, nil
}

func (r *DemoPurchaseRepository) HasPurchase(_ context.Context, userID, courseID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.purchases[demoPurchaseKey(userID, courseID)]
	return ok, nil
}
