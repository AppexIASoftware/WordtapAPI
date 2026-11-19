package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

var (
	ErrOrderNotFound             = errors.New("order not found")
	ErrPaymentAmountMismatch     = errors.New("payment amount does not match order")
	ErrPaymentCurrencyMismatch   = errors.New("payment currency must be PEN")
	ErrCourseAlreadyOwned        = errors.New("student already owns this course")
	ErrCourseNotPurchasableReal  = errors.New("course is not available for purchase")
	ErrBillingIdempotencyConflict = errors.New("idempotency key belongs to another item")
	ErrBillingStudentRequired    = errors.New("active student account is required")
)

type WebhookEventParams struct {
	Provider           string
	ProviderEventID    string
	EventType          string
	OrderID            string
	ProviderCheckoutID string
	ProviderPaymentID  string
	Status             entities.PaymentStatus
	AmountCents        int
	Currency           string
	RawPayload         string
}

type WebhookProcessResult struct {
	AlreadyProcessed bool
	OrderID          string
	Status           entities.PaymentStatus
}

type StudentEntitlementSummary struct {
	HasActiveSubscription bool     `json:"has_active_subscription"`
	Currency              string   `json:"currency"`
	CourseIDs             []string `json:"course_ids"`
}

type AdminMonetizationCourseRow struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	Level        string              `json:"level"`
	AccessTier   entities.AccessTier `json:"access_tier"`
	PriceCents   int                 `json:"price_cents"`
	Currency     string              `json:"currency"`
	Sales        int64               `json:"sales"`
	RevenueCents int64               `json:"revenue_cents"`
	AuthorName   string              `json:"author_name"`
}

type MySQLBillingRepository struct{ db *gorm.DB }

func NewMySQLBillingRepository(db *gorm.DB) *MySQLBillingRepository {
	return &MySQLBillingRepository{db: db}
}

func (r *MySQLBillingRepository) HasActiveSubscription(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	now := time.Now().UTC()
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.UserEntitlement{}).
		Where("user_id = ? AND entitlement_type = ? AND is_active = ? AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)",
			userID, entities.ProductTypeMonthlySubscription, true, now, now).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	err := r.db.WithContext(ctx).Model(&entities.Subscription{}).
		Where("user_id = ? AND status IN ? AND (current_period_end IS NULL OR current_period_end > ?)",
			userID, []string{"active", "trialing"}, now).
		Count(&count).Error
	return count > 0, err
}

func (r *MySQLBillingRepository) HasCourseEntitlement(ctx context.Context, userID, courseID string) (bool, error) {
	return hasCourseEntitlementTx(r.db.WithContext(ctx), userID, courseID, time.Now().UTC())
}

func hasCourseEntitlementTx(db *gorm.DB, userID, courseID string, now time.Time) (bool, error) {
	if userID == "" || courseID == "" {
		return false, nil
	}
	var count int64
	err := db.Model(&entities.UserEntitlement{}).
		Where("user_id = ? AND is_active = ? AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?) AND (course_id = ? OR product_id IN (SELECT id FROM products WHERE course_id = ?))",
			userID, true, now, now, courseID, courseID).
		Count(&count).Error
	return count > 0, err
}

func (r *MySQLBillingRepository) ListStudentEntitlements(ctx context.Context, userID string) (*StudentEntitlementSummary, error) {
	sub, err := r.HasActiveSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var ids []string
	if err := r.db.WithContext(ctx).
		Table("user_entitlements ue").
		Joins("LEFT JOIN products p ON p.id = ue.product_id").
		Where("ue.user_id = ? AND ue.is_active = ? AND ue.starts_at <= ? AND (ue.ends_at IS NULL OR ue.ends_at > ?) AND COALESCE(ue.course_id, p.course_id) IS NOT NULL",
			userID, true, now, now).
		Distinct().
		Pluck("COALESCE(ue.course_id, p.course_id)", &ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return &StudentEntitlementSummary{
		HasActiveSubscription: sub,
		Currency:              entities.CurrencyPEN,
		CourseIDs:             ids,
	}, nil
}

func (r *MySQLBillingRepository) CreateCourseOrder(ctx context.Context, userID, courseID, provider, idempotencyKey string) (*entities.Order, error) {
	provider = normalizeProvider(provider)
	key := strings.TrimSpace(idempotencyKey)
	var out entities.Order
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockActiveStudent(tx, userID); err != nil {
			return err
		}
		var course entities.Course
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&course, "id = ?", courseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCourseNotPurchasableReal
			}
			return err
		}
		if course.OwnerType != entities.OwnerTypeTeacher || course.Status != entities.ContentStatusPublished || course.AccessTier != entities.AccessTierCourse || course.PriceCents <= 0 {
			return ErrCourseNotPurchasableReal
		}
		if key != "" {
			var existing entities.Order
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Product").Where("user_id = ? AND idempotency_key = ?", userID, key).First(&existing).Error
			if err == nil {
				if existing.Product == nil || existing.Product.CourseID == nil || *existing.Product.CourseID != courseID ||
					!strings.EqualFold(existing.Provider, provider) || existing.AmountCents != course.PriceCents {
					return ErrBillingIdempotencyConflict
				}
				out = existing
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		owned, err := hasCourseEntitlementTx(tx, userID, courseID, time.Now().UTC())
		if err != nil {
			return err
		}
		if owned {
			return ErrCourseAlreadyOwned
		}
		prod, err := findOrUpsertProductTx(tx, &course.ID, course.Title, entities.ProductTypeOneTimeCourse, provider, course.PriceCents, entities.AccessTierCourse)
		if err != nil {
			return err
		}
		out = newPendingOrder(userID, prod.ID, provider, course.PriceCents, key)
		return tx.Create(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *MySQLBillingRepository) CreateSubscriptionOrder(ctx context.Context, userID, provider string, priceCents int, idempotencyKey string) (*entities.Order, error) {
	if priceCents < 100 {
		return nil, ErrCourseNotPurchasableReal
	}
	provider = normalizeProvider(provider)
	key := strings.TrimSpace(idempotencyKey)
	var out entities.Order
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockActiveStudent(tx, userID); err != nil {
			return err
		}
		if key != "" {
			var existing entities.Order
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Product").Where("user_id = ? AND idempotency_key = ?", userID, key).First(&existing).Error
			if err == nil {
				if existing.Product == nil || existing.Product.ProductType != entities.ProductTypeMonthlySubscription ||
					!strings.EqualFold(existing.Provider, provider) || existing.AmountCents != priceCents {
					return ErrBillingIdempotencyConflict
				}
				out = existing
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		prod, err := findOrUpsertProductTx(tx, nil, "Suscripción mensual WordTap", entities.ProductTypeMonthlySubscription, provider, priceCents, entities.AccessTierSubscription)
		if err != nil {
			return err
		}
		out = newPendingOrder(userID, prod.ID, provider, priceCents, key)
		return tx.Create(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
