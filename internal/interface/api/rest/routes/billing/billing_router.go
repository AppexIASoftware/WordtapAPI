package billing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	settingCmd "github.com/AppexIASoftware/WordtapAPI/internal/application/features/platform_setting/commands"
	"github.com/AppexIASoftware/WordtapAPI/internal/domain/entities"
	domainRepo "github.com/AppexIASoftware/WordtapAPI/internal/domain/repositories"
	infraRepo "github.com/AppexIASoftware/WordtapAPI/internal/infrastructure/repositories"
	"github.com/AppexIASoftware/WordtapAPI/internal/interface/api/rest/response"
)

const (
	defaultSubscriptionPriceCents = 2990
	maxWebhookBodyBytes           = 256 * 1024
)

type BillingStore interface {
	CreateCourseOrder(ctx context.Context, userID, courseID, provider, idempotencyKey string) (*entities.Order, error)
	CreateSubscriptionOrder(ctx context.Context, userID, provider string, priceCents int, idempotencyKey string) (*entities.Order, error)
	ListStudentEntitlements(ctx context.Context, userID string) (*infraRepo.StudentEntitlementSummary, error)
	ProcessPaymentWebhook(ctx context.Context, params infraRepo.WebhookEventParams) (*infraRepo.WebhookProcessResult, error)
}

type SimulatedCourseLister interface {
	ListPurchasedCourseIDs(ctx context.Context, userID string) ([]string, error)
}

type BillingRouter struct {
	store        BillingStore
	settingRepo  domainRepo.PlatformSettingRepository
	simLister    SimulatedCourseLister
	simEnabled   bool
	stripeSecret string
	mpSecret     string
	mpFetcher    MercadoPagoPaymentFetcher
	nowFn        func() time.Time
}

func NewBillingRouter(store BillingStore, settingRepo domainRepo.PlatformSettingRepository, stripeSecret, mpSecret string) *BillingRouter {
	return &BillingRouter{
		store:        store,
		settingRepo:  settingRepo,
		stripeSecret: stripeSecret,
		mpSecret:     mpSecret,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}

func (r *BillingRouter) WithSimulatedPurchases(lister SimulatedCourseLister, enabled bool) *BillingRouter {
	r.simLister = lister
	r.simEnabled = enabled
	return r
}

func (r *BillingRouter) WithMercadoPagoFetcher(fetcher MercadoPagoPaymentFetcher) *BillingRouter {
	r.mpFetcher = fetcher
	return r
}

func (r *BillingRouter) RegisterRoutes(
	v1 *echo.Group,
	authMiddleware echo.MiddlewareFunc,
	studentOnly echo.MiddlewareFunc,
	webhookLimiter echo.MiddlewareFunc,
) {
	group := v1.Group("/billing")
	group.Use(authMiddleware)
	group.GET("/entitlements/me", r.GetMyEntitlements)
	group.POST("/checkout/course/:courseId", r.CreateCourseCheckout, studentOnly)
	group.POST("/checkout/subscription", r.CreateSubscriptionCheckout, studentOnly)

	wh := v1.Group("/webhooks/payments")
	if webhookLimiter != nil {
		wh.POST("/:provider", r.HandleWebhook, webhookLimiter)
	} else {
		wh.POST("/:provider", r.HandleWebhook)
	}
}

type checkoutRequest struct {
	Provider string `json:"provider"`
}

func (r *BillingRouter) CreateCourseCheckout(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	courseID := strings.TrimSpace(c.Param("courseId"))
	if userID == "" || courseID == "" {
		return response.Fail(c, http.StatusBadRequest, "Curso o sesión inválida.")
	}
	provider, ok := parseCheckoutProvider(c)
	if !ok {
		return response.Fail(c, http.StatusBadRequest, "Pasarela de pago no soportada.")
	}
	order, err := r.store.CreateCourseOrder(c.Request().Context(), userID, courseID, provider, c.Request().Header.Get("Idempotency-Key"))
	if err != nil {
		return writeOrderError(c, err)
	}
	return c.JSON(http.StatusCreated, orderResponse(order))
}

func (r *BillingRouter) CreateSubscriptionCheckout(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	provider, ok := parseCheckoutProvider(c)
	if !ok {
		return response.Fail(c, http.StatusBadRequest, "Pasarela de pago no soportada.")
	}
	priceCents := resolveSubscriptionPriceCents(c.Request().Context(), r.settingRepo)
	order, err := r.store.CreateSubscriptionOrder(c.Request().Context(), userID, provider, priceCents, c.Request().Header.Get("Idempotency-Key"))
	if err != nil {
		return writeOrderError(c, err)
	}
	return c.JSON(http.StatusCreated, orderResponse(order))
}

func (r *BillingRouter) GetMyEntitlements(c *echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return response.Fail(c, http.StatusUnauthorized, "Sesión no válida.")
	}
	summary, err := r.store.ListStudentEntitlements(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "No se pudieron obtener los accesos.", err)
	}
	if r.simEnabled && r.simLister != nil {
		if simIDs, sErr := r.simLister.ListPurchasedCourseIDs(c.Request().Context(), userID); sErr == nil && len(simIDs) > 0 {
			seen := make(map[string]struct{}, len(summary.CourseIDs))
			for _, id := range summary.CourseIDs {
				seen[id] = struct{}{}
			}
			for _, id := range simIDs {
				if _, exists := seen[id]; !exists {
					summary.CourseIDs = append(summary.CourseIDs, id)
				}
			}
		}
	}
	return c.JSON(http.StatusOK, summary)
}

func (r *BillingRouter) HandleWebhook(c *echo.Context) error {
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	if provider != "stripe" && provider != "mercadopago" {
		return response.Fail(c, http.StatusBadRequest, "Proveedor de pago no soportado.")
	}
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxWebhookBodyBytes))
	if err != nil || len(body) == 0 {
		return response.Fail(c, http.StatusBadRequest, "Cuerpo de webhook inválido.")
	}

	params, ignored, parseErr := r.verifyAndParseWebhook(c, provider, body)
	if parseErr != nil {
		switch {
		case errors.Is(parseErr, ErrWebhookSecretMissing):
			return response.Fail(c, http.StatusServiceUnavailable, "Webhook no configurado en el servidor.")
		case errors.Is(parseErr, ErrWebhookSignatureInvalid):
			return response.Fail(c, http.StatusUnauthorized, "Firma de webhook inválida.")
		default:
			return response.Fail(c, http.StatusBadRequest, parseErr.Error())
		}
	}
	if ignored {
		return c.JSON(http.StatusOK, map[string]any{"received": true, "ignored": true})
	}
	res, err := r.store.ProcessPaymentWebhook(c.Request().Context(), params)
	if err != nil {
		switch {
		case errors.Is(err, infraRepo.ErrOrderNotFound):
			return response.Fail(c, http.StatusNotFound, "Orden de pago no encontrada.")
		case errors.Is(err, infraRepo.ErrPaymentCurrencyMismatch), errors.Is(err, infraRepo.ErrPaymentAmountMismatch), errors.Is(err, infraRepo.ErrOrderProviderMismatch):
			return response.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, infraRepo.ErrInvalidOrderTransition):
			return response.Fail(c, http.StatusConflict, err.Error())
		default:
			return response.Error(c, http.StatusInternalServerError, "Error procesando webhook de pago.", err)
		}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"received":          true,
		"already_processed": res.AlreadyProcessed,
		"order_id":          res.OrderID,
		"status":            res.Status,
	})
}

func (r *BillingRouter) verifyAndParseWebhook(c *echo.Context, provider string, body []byte) (infraRepo.WebhookEventParams, bool, error) {
	if provider == "stripe" {
		if err := VerifyStripeSignature(body, c.Request().Header.Get("Stripe-Signature"), r.stripeSecret, r.nowFn()); err != nil {
			return infraRepo.WebhookEventParams{}, false, err
		}
		var evt struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Data struct {
				Object struct {
					ID            string            `json:"id"`
					PaymentIntent string            `json:"payment_intent"`
					AmountTotal   int               `json:"amount_total"`
					Currency      string            `json:"currency"`
					Metadata      map[string]string `json:"metadata"`
				} `json:"object"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &evt); err != nil || strings.TrimSpace(evt.ID) == "" {
			return infraRepo.WebhookEventParams{}, false, errors.New("evento Stripe inválido")
		}
		status, known := mapWebhookStatus(evt.Type, "")
		if !known {
			return infraRepo.WebhookEventParams{}, true, nil
		}
		return infraRepo.WebhookEventParams{
			Provider:           "stripe",
			ProviderEventID:    strings.TrimSpace(evt.ID),
			EventType:          evt.Type,
			OrderID:            evt.Data.Object.Metadata["order_id"],
			ProviderCheckoutID: evt.Data.Object.ID,
			ProviderPaymentID:  evt.Data.Object.PaymentIntent,
			Status:             status,
			AmountCents:        evt.Data.Object.AmountTotal,
			Currency:           strings.ToUpper(strings.TrimSpace(evt.Data.Object.Currency)),
			RawPayload:         string(body),
		}, false, nil
	}

	var mp struct {
		ID     json.RawMessage `json:"id"`
		Type   string          `json:"type"`
		Action string          `json:"action"`
		Data   struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &mp); err != nil {
		return infraRepo.WebhookEventParams{}, false, errors.New("evento MercadoPago inválido")
	}
	qDataID := strings.TrimSpace(c.QueryParam("data.id"))
	bodyDataID := strings.TrimSpace(mp.Data.ID)
	if qDataID != "" && bodyDataID != "" && qDataID != bodyDataID {
		return infraRepo.WebhookEventParams{}, false, errors.New("data.id inconsistente")
	}
	dataID := firstNonEmpty(qDataID, bodyDataID)
	if dataID == "" {
		return infraRepo.WebhookEventParams{}, false, errors.New("data.id de MercadoPago requerido")
	}
	if err := VerifyMercadoPagoSignature(dataID, c.Request().Header.Get("x-request-id"), c.Request().Header.Get("x-signature"), r.mpSecret, r.nowFn()); err != nil {
		return infraRepo.WebhookEventParams{}, false, err
	}
	if r.mpFetcher == nil {
		return infraRepo.WebhookEventParams{}, false, ErrWebhookSecretMissing
	}
	verifiedPay, err := r.mpFetcher.FetchPayment(c.Request().Context(), dataID)
	if err != nil || verifiedPay == nil {
		return infraRepo.WebhookEventParams{}, false, errors.New("no se pudo verificar el pago en MercadoPago")
	}
	status, known := mapWebhookStatus(mp.Action, verifiedPay.Status)
	if !known {
		return infraRepo.WebhookEventParams{}, true, nil
	}
	evtID := strings.Trim(strings.TrimSpace(string(mp.ID)), `"`)
	if evtID == "" {
		evtID = firstNonEmpty(verifiedPay.ID, dataID)
	}
	return infraRepo.WebhookEventParams{
		Provider:          "mercadopago",
		ProviderEventID:   evtID,
		EventType:         firstNonEmpty(mp.Action, mp.Type, verifiedPay.Status),
		OrderID:           strings.TrimSpace(verifiedPay.ExternalReference),
		ProviderPaymentID: firstNonEmpty(verifiedPay.ID, dataID),
		Status:            status,
		AmountCents:       verifiedPay.AmountCents,
		Currency:          strings.ToUpper(strings.TrimSpace(verifiedPay.Currency)),
		RawPayload:        string(body),
	}, false, nil
}

func mapWebhookStatus(eventType, status string) (entities.PaymentStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "paid", "accredited":
		return entities.PaymentStatusApproved, true
	case "refunded", "charged_back":
		return entities.PaymentStatusRefunded, true
	case "cancelled":
		return entities.PaymentStatusCancelled, true
	case "rejected":
		return entities.PaymentStatusFailed, true
	}
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "checkout.session.completed", "payment_intent.succeeded", "invoice.payment_succeeded":
		return entities.PaymentStatusApproved, true
	case "charge.refunded":
		return entities.PaymentStatusRefunded, true
	case "checkout.session.expired":
		return entities.PaymentStatusCancelled, true
	case "payment_intent.payment_failed":
		return entities.PaymentStatusFailed, true
	}
	return "", false
}

func parseCheckoutProvider(c *echo.Context) (string, bool) {
	var req checkoutRequest
	_ = c.Bind(&req)
	p := strings.ToLower(strings.TrimSpace(req.Provider))
	if p == "" {
		return "mercadopago", true
	}
	if p != "mercadopago" && p != "stripe" {
		return "", false
	}
	return p, true
}

func writeOrderError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, infraRepo.ErrBillingStudentRequired):
		return response.Fail(c, http.StatusForbidden, "Solo los estudiantes activos pueden realizar compras.")
	case errors.Is(err, infraRepo.ErrCourseNotPurchasableReal):
		return response.Fail(c, http.StatusBadRequest, "El curso no está disponible para compra.")
	case errors.Is(err, infraRepo.ErrCourseAlreadyOwned), errors.Is(err, infraRepo.ErrBillingIdempotencyConflict):
		return response.Fail(c, http.StatusConflict, err.Error())
	default:
		return response.Error(c, http.StatusInternalServerError, "No se pudo iniciar la orden de pago.", err)
	}
}

func orderResponse(o *entities.Order) map[string]any {
	return map[string]any{
		"order_id":     o.ID,
		"product_id":   o.ProductID,
		"provider":     o.Provider,
		"amount_cents": o.AmountCents,
		"currency":     o.Currency,
		"status":       o.Status,
	}
}

func resolveSubscriptionPriceCents(ctx context.Context, repo domainRepo.PlatformSettingRepository) int {
	if repo == nil {
		return defaultSubscriptionPriceCents
	}
	s, err := repo.Get(ctx, settingCmd.SettingKeyMonetizationPlans)
	if err != nil || s == nil || s.Value == "" {
		return defaultSubscriptionPriceCents
	}
	var cfg struct {
		SubPrice float64 `json:"subPrice"`
	}
	if err := json.Unmarshal([]byte(s.Value), &cfg); err != nil || cfg.SubPrice <= 0 {
		return defaultSubscriptionPriceCents
	}
	return int(math.Round(cfg.SubPrice * 100))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
