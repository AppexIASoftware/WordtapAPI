package billing_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	infraRepo "github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/repositories"
	restMiddleware "github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/middleware"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/routes/billing"
)

type stubBillingStore struct {
	lastWebhook infraRepo.WebhookEventParams
	courses     []infraRepo.AdminMonetizationCourseRow
	lastTeacher string
	simIDs      []string
	order       *entities.Order
	orderErr    error
}

func (s *stubBillingStore) CreateCourseOrder(_ context.Context, userID, courseID, provider, _ string) (*entities.Order, error) {
	if s.orderErr != nil {
		return nil, s.orderErr
	}
	if s.order != nil {
		return s.order, nil
	}
	return &entities.Order{ID: "ord-1", UserID: userID, ProductID: "prod-" + courseID, Provider: provider, AmountCents: 2990, Currency: entities.CurrencyPEN, Status: entities.PaymentStatusPending}, nil
}

func (s *stubBillingStore) CreateSubscriptionOrder(_ context.Context, userID, provider string, priceCents int, _ string) (*entities.Order, error) {
	return &entities.Order{ID: "ord-sub", UserID: userID, ProductID: "prod-sub", Provider: provider, AmountCents: priceCents, Currency: entities.CurrencyPEN, Status: entities.PaymentStatusPending}, nil
}

func (s *stubBillingStore) ListStudentEntitlements(_ context.Context, _ string) (*infraRepo.StudentEntitlementSummary, error) {
	return &infraRepo.StudentEntitlementSummary{HasActiveSubscription: true, Currency: entities.CurrencyPEN, CourseIDs: []string{"c-real"}}, nil
}

func (s *stubBillingStore) ProcessPaymentWebhook(_ context.Context, params infraRepo.WebhookEventParams) (*infraRepo.WebhookProcessResult, error) {
	s.lastWebhook = params
	return &infraRepo.WebhookProcessResult{OrderID: params.OrderID, Status: params.Status}, nil
}

func (s *stubBillingStore) ListMonetizationOverviewCourses(_ context.Context, instructorID string) ([]infraRepo.AdminMonetizationCourseRow, error) {
	s.lastTeacher = instructorID
	return s.courses, nil
}

func (s *stubBillingStore) ListPurchasedCourseIDs(_ context.Context, _ string) ([]string, error) {
	return s.simIDs, nil
}

type stubSettings struct{ items []entities.PlatformSetting }

func (s *stubSettings) Get(_ context.Context, key string) (*entities.PlatformSetting, error) {
	for i := range s.items {
		if s.items[i].Key == key {
			return &s.items[i], nil
		}
	}
	return nil, nil
}
func (s *stubSettings) Set(context.Context, string, string) error              { return nil }
func (s *stubSettings) GetAll(context.Context) ([]entities.PlatformSetting, error) { return s.items, nil }

func signStripe(payload []byte, secret string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.", ts)
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func signMP(dataID, reqID, secret, ts string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "id:%s;request-id:%s;ts:%s;", dataID, reqID, ts)
	return fmt.Sprintf("ts=%s,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

type stubMPFetcher struct {
	payment *billing.MercadoPagoPayment
	err     error
}

func (f *stubMPFetcher) FetchPayment(_ context.Context, _ string) (*billing.MercadoPagoPayment, error) {
	return f.payment, f.err
}

func setupBillingTestEcho(store *stubBillingStore, settings *stubSettings, stripeSec, mpSec string, mpFetcher billing.MercadoPagoPaymentFetcher) *echo.Echo {
	e := echo.New()
	v1 := e.Group("/api/v1")
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			role := c.Request().Header.Get("X-Test-Role")
			if role == "" {
				role = string(entities.RoleStudent)
			}
			userID := c.Request().Header.Get("X-Test-User")
			if userID == "" && c.Request().Header.Get("X-Test-Empty-User") == "" {
				userID = "usr-1"
			}
			c.Set("user_id", userID)
			c.Set("user_role", role)
			return next(c)
		}
	}
	br := billing.NewBillingRouter(store, settings, stripeSec, mpSec).
		WithSimulatedPurchases(store, true).
		WithMercadoPagoFetcher(mpFetcher)
	br.RegisterRoutes(v1, auth, restMiddleware.RequireRole(entities.RoleStudent), nil)
	mr := billing.NewMonetizationAdminRouter(store, settings)
	mr.RegisterRoutes(v1, auth, restMiddleware.RequireRole(entities.RoleInstructor, entities.RoleAdmin))
	return e
}

func TestBillingRouter_CheckoutAndEntitlementsInPEN(t *testing.T) {
	store := &stubBillingStore{simIDs: []string{"c-sim"}}
	settings := &stubSettings{items: []entities.PlatformSetting{
		{Key: "monetization_plans_config", Value: `{"subPrice":34.9,"currency":"PEN"}`},
	}}
	e := setupBillingTestEcho(store, settings, "whsec_test", "mpsec_test", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout/course/c-1", bytes.NewBufferString(`{"provider":"mercadopago"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var res map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["currency"] != "PEN" || res["amount_cents"] != float64(2990) {
		t.Fatalf("unexpected course checkout payload: %+v", res)
	}

	reqSub := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout/subscription", bytes.NewBufferString(`{}`))
	reqSub.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recSub := httptest.NewRecorder()
	e.ServeHTTP(recSub, reqSub)
	if recSub.Code != http.StatusCreated {
		t.Fatalf("expected 201 for sub checkout, got %d", recSub.Code)
	}
	_ = json.Unmarshal(recSub.Body.Bytes(), &res)
	if res["amount_cents"] != float64(3490) || res["currency"] != "PEN" {
		t.Fatalf("expected 3490 PEN subscription order, got %+v", res)
	}

	recEnt := httptest.NewRecorder()
	e.ServeHTTP(recEnt, httptest.NewRequest(http.MethodGet, "/api/v1/billing/entitlements/me", nil))
	if recEnt.Code != http.StatusOK {
		t.Fatalf("expected 200 entitlements, got %d", recEnt.Code)
	}
	var ent infraRepo.StudentEntitlementSummary
	_ = json.Unmarshal(recEnt.Body.Bytes(), &ent)
	if !ent.HasActiveSubscription || ent.Currency != "PEN" || len(ent.CourseIDs) != 2 {
		t.Fatalf("expected merged PEN entitlements, got %+v", ent)
	}
}

func TestBillingRouter_HMACWebhooks(t *testing.T) {
	store := &stubBillingStore{}
	mpFetcher := &stubMPFetcher{
		payment: &billing.MercadoPagoPayment{
			ID:                "998877",
			ExternalReference: "ord-mp-verified",
			Status:            "approved",
			Currency:          "PEN",
			AmountCents:       2990,
		},
	}
	e := setupBillingTestEcho(store, &stubSettings{}, "whsec_test", "mpsec_test", mpFetcher)
	now := time.Now().UTC().Unix()

	stripeBody := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_1","payment_intent":"pi_1","amount_total":2990,"currency":"pen","metadata":{"order_id":"ord-1"}}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payments/stripe", bytes.NewReader(stripeBody))
	req.Header.Set("Stripe-Signature", signStripe(stripeBody, "whsec_test", now))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || store.lastWebhook.Status != entities.PaymentStatusApproved || store.lastWebhook.Currency != "PEN" {
		t.Fatalf("expected Stripe webhook 200 approved PEN, code=%d body=%s last=%+v", rec.Code, rec.Body.String(), store.lastWebhook)
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payments/stripe", bytes.NewReader(stripeBody))
	badReq.Header.Set("Stripe-Signature", signStripe(stripeBody, "wrong_secret", now))
	recBad := httptest.NewRecorder()
	e.ServeHTTP(recBad, badReq)
	if recBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad Stripe HMAC, got %d", recBad.Code)
	}

	// Body contains spoofed reference & amount, but verified server-to-server lookup wins
	mpBody := []byte(`{"id":"mp-evt-1","action":"payment.updated","data":{"id":"998877"},"external_reference":"tampered","transaction_amount":0.01}`)
	tsStr := strconv.FormatInt(now, 10)
	mpReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payments/mercadopago?data.id=998877", bytes.NewReader(mpBody))
	mpReq.Header.Set("x-request-id", "req-abc")
	mpReq.Header.Set("x-signature", signMP("998877", "req-abc", "mpsec_test", tsStr))
	recMP := httptest.NewRecorder()
	e.ServeHTTP(recMP, mpReq)
	if recMP.Code != http.StatusOK || store.lastWebhook.OrderID != "ord-mp-verified" || store.lastWebhook.AmountCents != 2990 {
		t.Fatalf("expected verified MP payment ord-mp-verified 2990 PEN, code=%d body=%s last=%+v", recMP.Code, recMP.Body.String(), store.lastWebhook)
	}

	mismatchReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payments/mercadopago?data.id=111111", bytes.NewReader(mpBody))
	mismatchReq.Header.Set("x-request-id", "req-abc")
	mismatchReq.Header.Set("x-signature", signMP("111111", "req-abc", "mpsec_test", tsStr))
	recMis := httptest.NewRecorder()
	e.ServeHTTP(recMis, mismatchReq)
	if recMis.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on query/body data.id mismatch, got %d", recMis.Code)
	}

	eNoFetcher := setupBillingTestEcho(store, &stubSettings{}, "whsec_test", "mpsec_test", nil)
	mpNoFetchReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payments/mercadopago?data.id=998877", bytes.NewReader(mpBody))
	mpNoFetchReq.Header.Set("x-request-id", "req-abc")
	mpNoFetchReq.Header.Set("x-signature", signMP("998877", "req-abc", "mpsec_test", tsStr))
	recNoFetch := httptest.NewRecorder()
	eNoFetcher.ServeHTTP(recNoFetch, mpNoFetchReq)
	if recNoFetch.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when MP access token fetcher is not configured, got %d", recNoFetch.Code)
	}

	eNoSec := setupBillingTestEcho(store, &stubSettings{}, "", "", nil)
	recNoSec := httptest.NewRecorder()
	eNoSec.ServeHTTP(recNoSec, httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/payments/mercadopago", bytes.NewReader(mpBody)))
	if recNoSec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 fail-closed when secret missing, got %d", recNoSec.Code)
	}
}

func TestMonetizationAdminRouter_OverviewAndRBAC(t *testing.T) {
	store := &stubBillingStore{
		courses: []infraRepo.AdminMonetizationCourseRow{{ID: "c-1", Title: "Inglés IT", Level: "B1", AccessTier: entities.AccessTierCourse, PriceCents: 2990, Currency: entities.CurrencyPEN, Sales: 4, RevenueCents: 11960}},
	}
	settings := &stubSettings{items: []entities.PlatformSetting{
		{Key: "course_min_price_pen", Value: "2500"},
	}}
	e := setupBillingTestEcho(store, settings, "", "", nil)

	reqEmpty := httptest.NewRequest(http.MethodGet, "/api/v1/admin/monetization/overview", nil)
	reqEmpty.Header.Set("X-Test-Role", string(entities.RoleInstructor))
	reqEmpty.Header.Set("X-Test-Empty-User", "1")
	recEmpty := httptest.NewRecorder()
	e.ServeHTTP(recEmpty, reqEmpty)
	if recEmpty.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user_id is empty, got %d", recEmpty.Code)
	}

	reqStu := httptest.NewRequest(http.MethodGet, "/api/v1/admin/monetization/overview", nil)
	reqStu.Header.Set("X-Test-Role", string(entities.RoleStudent))
	recStu := httptest.NewRecorder()
	e.ServeHTTP(recStu, reqStu)
	if recStu.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for student on overview, got %d", recStu.Code)
	}

	reqTch := httptest.NewRequest(http.MethodGet, "/api/v1/admin/monetization/overview", nil)
	reqTch.Header.Set("X-Test-Role", string(entities.RoleInstructor))
	recTch := httptest.NewRecorder()
	e.ServeHTTP(recTch, reqTch)
	if recTch.Code != http.StatusOK || store.lastTeacher != "usr-1" {
		t.Fatalf("expected instructor-scoped overview, code=%d teacher=%q", recTch.Code, store.lastTeacher)
	}
	var tchResp map[string]any
	_ = json.Unmarshal(recTch.Body.Bytes(), &tchResp)
	plans := tchResp["plans_config"].(map[string]any)
	if plans["gateways"] != nil || plans["courseMin"] != float64(25) || tchResp["currency"] != "PEN" {
		t.Fatalf("expected no gateways and courseMin=25 PEN for instructor, got %+v", tchResp)
	}

	reqAdm := httptest.NewRequest(http.MethodGet, "/api/v1/admin/monetization/overview", nil)
	reqAdm.Header.Set("X-Test-Role", string(entities.RoleAdmin))
	recAdm := httptest.NewRecorder()
	e.ServeHTTP(recAdm, reqAdm)
	if recAdm.Code != http.StatusOK || store.lastTeacher != "" {
		t.Fatalf("expected unscoped admin overview, code=%d teacher=%q", recAdm.Code, store.lastTeacher)
	}
}
