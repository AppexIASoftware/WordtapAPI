package repositories

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
)

func TestMySQLBillingRepository_EntitlementsAndSubscriptions(t *testing.T) {
	t.Run("HasActiveSubscription checks unexpired user_entitlements/subscriptions and never users.access_tier", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`users`") {
					t.Fatalf("HasActiveSubscription must not trust users.access_tier lifetime column: %s", q)
				}
				if strings.Contains(q, "`user_entitlements`") {
					return queryResponse{columns: []string{"count"}, rows: [][]driver.Value{{int64(0)}}}, true
				}
				if strings.Contains(q, "`subscriptions`") {
					return queryResponse{columns: []string{"count"}, rows: [][]driver.Value{{int64(1)}}}, true
				}
				return queryResponse{columns: []string{"count"}, rows: [][]driver.Value{{int64(0)}}}, true
			},
		}
		repo := NewMySQLBillingRepository(newLockingTestGormDB(t, state))
		ok, err := repo.HasActiveSubscription(context.Background(), "sub-user-1")
		if err != nil || !ok {
			t.Fatalf("expected active subscription=true, err=%v", err)
		}
	})

	t.Run("HasCourseEntitlement checks active user_entitlements", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`user_entitlements`") {
					return queryResponse{columns: []string{"count"}, rows: [][]driver.Value{{int64(1)}}}, true
				}
				return queryResponse{columns: []string{"count"}, rows: [][]driver.Value{{int64(0)}}}, true
			},
		}
		repo := NewMySQLBillingRepository(newLockingTestGormDB(t, state))
		ok, err := repo.HasCourseEntitlement(context.Background(), "stu-1", "course-1")
		if err != nil || !ok {
			t.Fatalf("expected course entitlement=true, err=%v", err)
		}
	})
}

func TestMySQLBillingRepository_CreateCourseOrderAndWebhooks(t *testing.T) {
	now := time.Now().UTC()

	t.Run("CreateCourseOrder locks student & course FOR UPDATE and writes PEN order", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				switch {
				case strings.Contains(q, "`users`"):
					return queryResponse{columns: []string{"id", "role", "is_active"}, rows: [][]driver.Value{{"stu-1", "student", true}}}, true
				case strings.Contains(q, "`courses`"):
					return queryResponse{
						columns: []string{"id", "owner_type", "status", "access_tier", "price_cents", "currency", "title"},
						rows:    [][]driver.Value{{"crs-1", "teacher", "published", "course", int64(2990), "PEN", "Curso Soles"}},
					}, true
				case strings.Contains(strings.ToLower(q), "count(*)"):
					return queryResponse{columns: []string{"count"}, rows: [][]driver.Value{{int64(0)}}}, true
				case strings.Contains(q, "`products`"):
					return queryResponse{
						columns: []string{"id", "course_id", "price_cents", "currency", "product_type"},
						rows:    [][]driver.Value{{"prod-1", "crs-1", int64(2990), "PEN", "one_time_course"}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		repo := NewMySQLBillingRepository(newLockingTestGormDB(t, state))
		order, err := repo.CreateCourseOrder(context.Background(), "stu-1", "crs-1", "mercadopago", "idem-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if order.Currency != entities.CurrencyPEN || order.AmountCents != 2990 || order.Provider != "mercadopago" {
			t.Fatalf("expected 2990 PEN order via mercadopago, got %+v", order)
		}
		if !hasQueryContaining(state.queries, "FOR UPDATE") || !hasPrefixEvent(state.events, "COMMIT") {
			t.Fatalf("expected FOR UPDATE and COMMIT, events=%v", state.events)
		}
	})

	t.Run("ProcessPaymentWebhook enforces PEN, amount, provider, non-resurrection, and idempotency", func(t *testing.T) {
		repoBad := NewMySQLBillingRepository(newLockingTestGormDB(t, &mockDBState{}))
		if _, err := repoBad.ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: "evt-usd",
			OrderID:         "ord-1",
			Status:          entities.PaymentStatusApproved,
			AmountCents:     2990,
			Currency:        "USD",
		}); !errors.Is(err, ErrPaymentCurrencyMismatch) {
			t.Fatalf("expected ErrPaymentCurrencyMismatch, got %v", err)
		}
		if _, err := repoBad.ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: "evt-empty-cur",
			OrderID:         "ord-1",
			Status:          entities.PaymentStatusApproved,
			AmountCents:     2990,
			Currency:        "",
		}); !errors.Is(err, ErrPaymentCurrencyMismatch) {
			t.Fatalf("expected ErrPaymentCurrencyMismatch on empty approved currency, got %v", err)
		}
		if _, err := repoBad.ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: "evt-zero-amt",
			OrderID:         "ord-1",
			Status:          entities.PaymentStatusApproved,
			AmountCents:     0,
			Currency:        "PEN",
		}); !errors.Is(err, ErrPaymentAmountMismatch) {
			t.Fatalf("expected ErrPaymentAmountMismatch on 0 approved amount, got %v", err)
		}

		crossProvState := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`orders`") {
					return queryResponse{
						columns: []string{"id", "user_id", "product_id", "provider", "amount_cents", "currency", "status"},
						rows:    [][]driver.Value{{"ord-1", "stu-1", "prod-1", "mercadopago", int64(2990), "PEN", "pending"}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		if _, err := NewMySQLBillingRepository(newLockingTestGormDB(t, crossProvState)).ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: "evt-cross",
			OrderID:         "ord-1",
			Status:          entities.PaymentStatusApproved,
			AmountCents:     2990,
			Currency:        "PEN",
		}); !errors.Is(err, ErrOrderProviderMismatch) {
			t.Fatalf("expected ErrOrderProviderMismatch, got %v", err)
		}

		refundedState := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`orders`") {
					return queryResponse{
						columns: []string{"id", "user_id", "product_id", "provider", "amount_cents", "currency", "status"},
						rows:    [][]driver.Value{{"ord-1", "stu-1", "prod-1", "stripe", int64(2990), "PEN", "refunded"}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		if _, err := NewMySQLBillingRepository(newLockingTestGormDB(t, refundedState)).ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: "evt-resurrect",
			OrderID:         "ord-1",
			Status:          entities.PaymentStatusApproved,
			AmountCents:     2990,
			Currency:        "PEN",
		}); !errors.Is(err, ErrInvalidOrderTransition) {
			t.Fatalf("expected ErrInvalidOrderTransition blocking post-refund approval, got %v", err)
		}

		dupState := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				if strings.Contains(q, "`payment_events`") {
					return queryResponse{columns: []string{"id", "provider_event_id", "processed_at"}, rows: [][]driver.Value{{"pe-1", "evt-dup", now}}}, true
				}
				return queryResponse{}, false
			},
		}
		dupRes, err := NewMySQLBillingRepository(newLockingTestGormDB(t, dupState)).ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:        "mercadopago",
			ProviderEventID: "evt-dup",
			OrderID:         "ord-1",
			Status:          entities.PaymentStatusApproved,
			AmountCents:     2990,
			Currency:        "PEN",
		})
		if err != nil || !dupRes.AlreadyProcessed {
			t.Fatalf("expected idempotent AlreadyProcessed=true, got %+v err=%v", dupRes, err)
		}

		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				switch {
				case strings.Contains(q, "`orders`"):
					return queryResponse{
						columns: []string{"id", "user_id", "product_id", "provider", "amount_cents", "currency", "status"},
						rows:    [][]driver.Value{{"ord-1", "stu-1", "prod-1", "mercadopago", int64(2990), "PEN", "pending"}},
					}, true
				case strings.Contains(q, "`products`"):
					return queryResponse{
						columns: []string{"id", "course_id", "product_type"},
						rows:    [][]driver.Value{{"prod-1", "crs-1", "one_time_course"}},
					}, true
				}
				return queryResponse{}, false
			},
		}
		res, err := NewMySQLBillingRepository(newLockingTestGormDB(t, state)).ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:          "mercadopago",
			ProviderEventID:   "evt-ok-1",
			EventType:         "payment.created",
			OrderID:           "ord-1",
			ProviderPaymentID: "mp-pay-99",
			Status:            entities.PaymentStatusApproved,
			AmountCents:       2990,
			Currency:          "PEN",
		})
		if err != nil || res.AlreadyProcessed || res.Status != entities.PaymentStatusApproved {
			t.Fatalf("unexpected webhook approval result: %+v err=%v", res, err)
		}
		if len(state.execs) < 3 || !hasPrefixEvent(state.events, "COMMIT") {
			t.Fatalf("expected order update + entitlement insert + event insert + COMMIT, execs=%d events=%v", len(state.execs), state.events)
		}
	})

	t.Run("ProcessPaymentWebhook revokes subscription entitlement, subscriptions, and access_tier on refund", func(t *testing.T) {
		state := &mockDBState{
			onQuery: func(q string, _ []driver.Value) (queryResponse, bool) {
				switch {
				case strings.Contains(q, "`orders`"):
					return queryResponse{
						columns: []string{"id", "user_id", "product_id", "provider", "amount_cents", "currency", "status"},
						rows:    [][]driver.Value{{"ord-sub", "stu-1", "prod-sub", "stripe", int64(2990), "PEN", "approved"}},
					}, true
				case strings.Contains(q, "`products`"):
					return queryResponse{
						columns: []string{"id", "course_id", "product_type"},
						rows:    [][]driver.Value{{"prod-sub", nil, "monthly_subscription"}},
					}, true
				case strings.Contains(strings.ToLower(q), "count(*)"):
					return queryResponse{columns: []string{"count"}, rows: [][]driver.Value{{int64(0)}}}, true
				}
				return queryResponse{}, false
			},
		}
		res, err := NewMySQLBillingRepository(newLockingTestGormDB(t, state)).ProcessPaymentWebhook(context.Background(), WebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: "evt-refund-1",
			EventType:       "charge.refunded",
			OrderID:         "ord-sub",
			Status:          entities.PaymentStatusRefunded,
			Currency:        "PEN",
		})
		if err != nil || res.Status != entities.PaymentStatusRefunded {
			t.Fatalf("unexpected refund result: %+v err=%v", res, err)
		}
		revertedUser, cancelledSub := false, false
		for _, ex := range state.execs {
			if strings.Contains(ex.sql, "UPDATE `users`") {
				revertedUser = true
			}
			if strings.Contains(ex.sql, "UPDATE `subscriptions`") {
				cancelledSub = true
			}
		}
		if !revertedUser || !cancelledSub {
			t.Fatalf("expected subscription refund to cancel subscriptions (%v) and revert user access_tier (%v)", cancelledSub, revertedUser)
		}
	})
}
