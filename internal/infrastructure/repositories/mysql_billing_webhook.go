package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

var (
	ErrOrderProviderMismatch  = errors.New("webhook provider does not match order")
	ErrInvalidOrderTransition = errors.New("invalid order state transition")
)

func (r *MySQLBillingRepository) ProcessPaymentWebhook(ctx context.Context, params WebhookEventParams) (*WebhookProcessResult, error) {
	curr := strings.TrimSpace(params.Currency)
	if params.Status == entities.PaymentStatusApproved {
		if !strings.EqualFold(curr, entities.CurrencyPEN) {
			return nil, ErrPaymentCurrencyMismatch
		}
		if params.AmountCents <= 0 {
			return nil, ErrPaymentAmountMismatch
		}
	} else if curr != "" && !strings.EqualFold(curr, entities.CurrencyPEN) {
		return nil, ErrPaymentCurrencyMismatch
	}
	provider := normalizeProvider(params.Provider)
	var out WebhookProcessResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingEvt entities.PaymentEvent
		evtErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("provider_event_id = ?", params.ProviderEventID).
			First(&existingEvt).Error
		if evtErr == nil && existingEvt.ProcessedAt != nil {
			out = WebhookProcessResult{
				AlreadyProcessed: true,
				OrderID:          params.OrderID,
				Status:           params.Status,
			}
			return nil
		}
		if evtErr != nil && !errors.Is(evtErr, gorm.ErrRecordNotFound) {
			return evtErr
		}

		var order entities.Order
		orderQuery := tx.Clauses(clause.Locking{Strength: "UPDATE"})
		switch {
		case strings.TrimSpace(params.OrderID) != "":
			orderQuery = orderQuery.Where("id = ?", strings.TrimSpace(params.OrderID))
		case strings.TrimSpace(params.ProviderCheckoutID) != "":
			orderQuery = orderQuery.Where("provider_checkout_id = ?", strings.TrimSpace(params.ProviderCheckoutID))
		default:
			return ErrOrderNotFound
		}
		if err := orderQuery.First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		if !strings.EqualFold(order.Provider, provider) {
			return ErrOrderProviderMismatch
		}
		if !strings.EqualFold(order.Currency, entities.CurrencyPEN) {
			return ErrPaymentCurrencyMismatch
		}
		if params.Status == entities.PaymentStatusApproved {
			if params.AmountCents != order.AmountCents {
				return ErrPaymentAmountMismatch
			}
			if order.Status != entities.PaymentStatusPending && order.Status != entities.PaymentStatusApproved {
				return ErrInvalidOrderTransition
			}
		} else if params.AmountCents != 0 && params.AmountCents != order.AmountCents {
			return ErrPaymentAmountMismatch
		}

		var prod entities.Product
		if err := tx.First(&prod, "id = ?", order.ProductID).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		switch params.Status {
		case entities.PaymentStatusApproved:
			if err := applyOrderApprovalTx(tx, &order, &prod, params.ProviderPaymentID, now); err != nil {
				return err
			}
		case entities.PaymentStatusRefunded, entities.PaymentStatusCancelled:
			if err := applyOrderRevocationTx(tx, &order, &prod, params.Status, now); err != nil {
				return err
			}
		case entities.PaymentStatusFailed:
			if order.Status == entities.PaymentStatusPending {
				if err := tx.Model(&order).Updates(map[string]any{
					"status":     entities.PaymentStatusFailed,
					"updated_at": now,
				}).Error; err != nil {
					return err
				}
				order.Status = entities.PaymentStatusFailed
			}
		}

		if errors.Is(evtErr, gorm.ErrRecordNotFound) {
			rawPayload := strings.TrimSpace(params.RawPayload)
			if rawPayload == "" {
				rawPayload = "{}"
			}
			evt := entities.PaymentEvent{
				ID:              uuid.NewString(),
				Provider:        provider,
				ProviderEventID: params.ProviderEventID,
				EventType:       params.EventType,
				Payload:         rawPayload,
				ProcessedAt:     &now,
				CreatedAt:       now,
			}
			if err := tx.Create(&evt).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&existingEvt).Update("processed_at", &now).Error; err != nil {
			return err
		}

		out = WebhookProcessResult{
			AlreadyProcessed: false,
			OrderID:          order.ID,
			Status:           order.Status,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func applyOrderApprovalTx(tx *gorm.DB, order *entities.Order, prod *entities.Product, paymentID string, now time.Time) error {
	wasApproved := order.Status == entities.PaymentStatusApproved
	updates := map[string]any{
		"status":     entities.PaymentStatusApproved,
		"paid_at":    &now,
		"updated_at": now,
	}
	if strings.TrimSpace(paymentID) != "" {
		pid := strings.TrimSpace(paymentID)
		updates["provider_payment_id"] = &pid
	}
	if err := tx.Model(order).Updates(updates).Error; err != nil {
		return err
	}
	order.Status = entities.PaymentStatusApproved
	if wasApproved {
		return nil
	}
	var endsAt *time.Time
	if prod.ProductType == entities.ProductTypeMonthlySubscription {
		end := now.Add(30 * 24 * time.Hour)
		endsAt = &end
	}
	ent := entities.UserEntitlement{
		ID:              uuid.NewString(),
		UserID:          order.UserID,
		ProductID:       prod.ID,
		CourseID:        prod.CourseID,
		OrderID:         &order.ID,
		EntitlementType: prod.ProductType,
		StartsAt:        now,
		EndsAt:          endsAt,
		IsActive:        true,
		CreatedAt:       now,
	}
	if err := tx.Create(&ent).Error; err != nil {
		return err
	}
	if prod.ProductType == entities.ProductTypeMonthlySubscription {
		return tx.Model(&entities.User{}).
			Where("id = ? AND access_tier = ?", order.UserID, entities.AccessTierFree).
			Updates(map[string]any{"access_tier": entities.AccessTierSubscription, "updated_at": now}).Error
	}
	return nil
}

func applyOrderRevocationTx(tx *gorm.DB, order *entities.Order, prod *entities.Product, status entities.PaymentStatus, now time.Time) error {
	if err := tx.Model(order).Updates(map[string]any{
		"status":     status,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}
	order.Status = status
	if err := tx.Model(&entities.UserEntitlement{}).
		Where("order_id = ? AND is_active = ?", order.ID, true).
		Update("is_active", false).Error; err != nil {
		return err
	}
	if prod.ProductType == entities.ProductTypeMonthlySubscription {
		if err := tx.Model(&entities.Subscription{}).
			Where("user_id = ? AND product_id = ? AND status IN ?", order.UserID, prod.ID, []string{"active", "trialing"}).
			Updates(map[string]any{"status": "cancelled", "cancelled_at": &now, "updated_at": now}).Error; err != nil {
			return err
		}
		var otherCount int64
		if err := tx.Model(&entities.UserEntitlement{}).
			Where("user_id = ? AND entitlement_type = ? AND is_active = ? AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)",
				order.UserID, entities.ProductTypeMonthlySubscription, true, now, now).
			Count(&otherCount).Error; err != nil {
			return err
		}
		if otherCount == 0 {
			return tx.Model(&entities.User{}).
				Where("id = ? AND access_tier = ?", order.UserID, entities.AccessTierSubscription).
				Updates(map[string]any{"access_tier": entities.AccessTierFree, "updated_at": now}).Error
		}
	}
	return nil
}

func (r *MySQLBillingRepository) ListMonetizationOverviewCourses(ctx context.Context, instructorID string) ([]AdminMonetizationCourseRow, error) {
	rows := make([]AdminMonetizationCourseRow, 0)
	query := r.db.WithContext(ctx).
		Table("courses c").
		Select(`c.id, c.title, c.level, c.access_tier, c.price_cents, c.currency,
			COALESCE(u.name, '') AS author_name,
			COUNT(o.id) AS sales,
			COALESCE(SUM(o.amount_cents), 0) AS revenue_cents`).
		Joins("LEFT JOIN users u ON u.id = c.created_by").
		Joins("LEFT JOIN products p ON p.course_id = c.id").
		Joins("LEFT JOIN orders o ON o.product_id = p.id AND o.status = ?", entities.PaymentStatusApproved).
		Where("c.owner_type = ?", entities.OwnerTypeTeacher).
		Group("c.id, c.title, c.level, c.access_tier, c.price_cents, c.currency, u.name").
		Order("revenue_cents DESC, c.created_at DESC")

	if strings.TrimSpace(instructorID) != "" {
		query = query.Where("c.created_by = ?", strings.TrimSpace(instructorID))
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		if strings.TrimSpace(rows[i].Currency) == "" {
			rows[i].Currency = entities.CurrencyPEN
		}
	}
	return rows, nil
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "stripe":
		return "stripe"
	default:
		return "mercadopago"
	}
}

func lockActiveStudent(tx *gorm.DB, userID string) error {
	var user entities.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "role", "is_active").First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBillingStudentRequired
		}
		return err
	}
	if user.Role != entities.RoleStudent || !user.IsActive {
		return ErrBillingStudentRequired
	}
	return nil
}

func findOrUpsertProductTx(tx *gorm.DB, courseID *string, name string, pType entities.ProductType, provider string, priceCents int, tier entities.AccessTier) (*entities.Product, error) {
	var prod entities.Product
	query := tx.Where("product_type = ? AND provider = ?", pType, provider)
	if courseID != nil {
		query = query.Where("course_id = ?", *courseID)
	} else {
		query = query.Where("course_id IS NULL")
	}
	err := query.First(&prod).Error
	if err == nil {
		if prod.PriceCents != priceCents || prod.Currency != entities.CurrencyPEN {
			if err := tx.Model(&prod).Updates(map[string]any{"price_cents": priceCents, "currency": entities.CurrencyPEN}).Error; err != nil {
				return nil, err
			}
			prod.PriceCents = priceCents
			prod.Currency = entities.CurrencyPEN
		}
		return &prod, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	prod = entities.Product{
		ID:          uuid.NewString(),
		CourseID:    courseID,
		Name:        name,
		ProductType: pType,
		Provider:    provider,
		PriceCents:  priceCents,
		Currency:    entities.CurrencyPEN,
		AccessTier:  tier,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
	}
	if err := tx.Create(&prod).Error; err != nil {
		return nil, err
	}
	return &prod, nil
}

func newPendingOrder(userID, productID, provider string, amountCents int, idempotencyKey string) entities.Order {
	now := time.Now().UTC()
	var keyPtr *string
	if idempotencyKey != "" {
		k := idempotencyKey
		keyPtr = &k
	}
	return entities.Order{
		ID:             uuid.NewString(),
		UserID:         userID,
		ProductID:      productID,
		Provider:       provider,
		AmountCents:    amountCents,
		Currency:       entities.CurrencyPEN,
		Status:         entities.PaymentStatusPending,
		IdempotencyKey: keyPtr,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
