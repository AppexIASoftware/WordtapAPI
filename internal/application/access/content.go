package access

import (
	"context"
	"errors"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

var ErrContentDenied = errors.New("course content access denied")

type PurchaseReader interface {
	HasPurchase(context.Context, string, string) (bool, error)
}

// ContentPolicy grants simulated entitlements only while simulation is enabled.
type ContentPolicy struct {
	purchases PurchaseReader
	enabled   bool
}

func NewContentPolicy(purchases PurchaseReader, enabled bool) *ContentPolicy {
	return &ContentPolicy{purchases: purchases, enabled: enabled}
}

func (p *ContentPolicy) Authorize(ctx context.Context, course *entities.Course, lesson *entities.Lesson, userID string, role entities.UserRole) error {
	if course == nil || userID == "" {
		return ErrContentDenied
	}
	if CanViewCourse(course, userID, role) {
		return nil
	}
	if course.Status != entities.ContentStatusPublished || (lesson != nil && (lesson.CourseID != course.ID || lesson.Status != entities.ContentStatusPublished)) {
		return ErrContentDenied
	}
	if EffectiveOwnerType(course) == entities.OwnerTypePlatform {
		return nil
	}
	if role != entities.RoleStudent || p == nil || !p.enabled || p.purchases == nil {
		return ErrContentDenied
	}
	purchase, err := p.purchases.HasPurchase(ctx, userID, course.ID)
	if err != nil {
		return err
	}
	if !purchase {
		return ErrContentDenied
	}
	return nil
}
