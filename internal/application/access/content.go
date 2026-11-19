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

type EntitlementReader interface {
	HasActiveSubscription(ctx context.Context, userID string) (bool, error)
	HasCourseEntitlement(ctx context.Context, userID, courseID string) (bool, error)
}

// ContentPolicy enforces server-side entitlements, allowing simulated purchases only when enabled.
type ContentPolicy struct {
	purchases    PurchaseReader
	entitlements EntitlementReader
	enabled      bool
}

func NewContentPolicy(purchases PurchaseReader, enabled bool) *ContentPolicy {
	return &ContentPolicy{purchases: purchases, enabled: enabled}
}

// WithEntitlements attaches the real billing entitlement reader (subscription & course purchases).
func (p *ContentPolicy) WithEntitlements(entitlements EntitlementReader) *ContentPolicy {
	p.entitlements = entitlements
	return p
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
	if role != entities.RoleStudent {
		return ErrContentDenied
	}
	if EffectiveOwnerType(course) == entities.OwnerTypePlatform {
		requiresSub := course.AccessTier == entities.AccessTierSubscription ||
			(lesson != nil && lesson.AccessTier == entities.AccessTierSubscription)
		if requiresSub {
			if p == nil || p.entitlements == nil {
				return ErrContentDenied
			}
			sub, err := p.entitlements.HasActiveSubscription(ctx, userID)
			if err != nil {
				return err
			}
			if !sub {
				return ErrContentDenied
			}
			return nil
		}
		requiresCourse := course.AccessTier == entities.AccessTierCourse ||
			(lesson != nil && lesson.AccessTier == entities.AccessTierCourse)
		if !requiresCourse {
			return nil
		}
	}
	if EffectiveOwnerType(course) == entities.OwnerTypeTeacher &&
		course.AccessTier == entities.AccessTierFree &&
		course.PriceCents == 0 &&
		(lesson == nil || lesson.AccessTier == entities.AccessTierFree || lesson.AccessTier == "") {
		return nil
	}
	if p != nil && p.entitlements != nil {
		entitled, err := p.entitlements.HasCourseEntitlement(ctx, userID, course.ID)
		if err != nil {
			return err
		}
		if entitled {
			return nil
		}
	}
	if p == nil || !p.enabled || p.purchases == nil {
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
