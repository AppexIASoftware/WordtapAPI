package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrWebhookSecretMissing    = errors.New("webhook secret is not configured")
	ErrWebhookSignatureInvalid = errors.New("invalid webhook signature")
)

type MercadoPagoPayment struct {
	ID                string
	ExternalReference string
	Status            string
	Currency          string
	AmountCents       int
}

type MercadoPagoPaymentFetcher interface {
	FetchPayment(ctx context.Context, paymentID string) (*MercadoPagoPayment, error)
}

type HTTPMercadoPagoClient struct {
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

func NewHTTPMercadoPagoClient(accessToken string) *HTTPMercadoPagoClient {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil
	}
	return &HTTPMercadoPagoClient{
		accessToken: token,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		baseURL:     "https://api.mercadopago.com",
	}
}

func (c *HTTPMercadoPagoClient) FetchPayment(ctx context.Context, paymentID string) (*MercadoPagoPayment, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/payments/"+url.PathEscape(strings.TrimSpace(paymentID)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mercadopago payment lookup failed: status %d", resp.StatusCode)
	}
	var payload struct {
		ID                json.RawMessage `json:"id"`
		ExternalReference string          `json:"external_reference"`
		Status            string          `json:"status"`
		CurrencyID        string          `json:"currency_id"`
		TransactionAmount float64         `json:"transaction_amount"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&payload); err != nil {
		return nil, err
	}
	return &MercadoPagoPayment{
		ID:                strings.Trim(strings.TrimSpace(string(payload.ID)), `"`),
		ExternalReference: strings.TrimSpace(payload.ExternalReference),
		Status:            strings.TrimSpace(payload.Status),
		Currency:          strings.ToUpper(strings.TrimSpace(payload.CurrencyID)),
		AmountCents:       int(math.Round(payload.TransactionAmount * 100)),
	}, nil
}

const maxWebhookReplayAge = 5 * time.Minute

// VerifyStripeSignature verifies Stripe's t=...,v1=... HMAC-SHA256 header.
func VerifyStripeSignature(payload []byte, header, secret string, now time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return ErrWebhookSecretMissing
	}
	tsStr, sigs := parseCommaKeyValues(header, "t", "v1")
	if tsStr == "" || len(sigs) == 0 {
		return ErrWebhookSignatureInvalid
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil || math.Abs(now.Sub(time.Unix(ts, 0).UTC()).Seconds()) > maxWebhookReplayAge.Seconds() {
		return ErrWebhookSignatureInvalid
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.", ts)
	mac.Write(payload)
	expected := mac.Sum(nil)
	for _, sig := range sigs {
		decoded, err := hex.DecodeString(sig)
		if err == nil && hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return ErrWebhookSignatureInvalid
}

// VerifyMercadoPagoSignature verifies MercadoPago's x-signature (ts=...,v1=...) manifest.
func VerifyMercadoPagoSignature(dataID, requestID, sigHeader, secret string, now time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return ErrWebhookSecretMissing
	}
	if strings.TrimSpace(dataID) == "" || strings.TrimSpace(requestID) == "" {
		return ErrWebhookSignatureInvalid
	}
	tsStr, sigs := parseCommaKeyValues(sigHeader, "ts", "v1")
	if tsStr == "" || len(sigs) == 0 {
		return ErrWebhookSignatureInvalid
	}
	tsNum, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return ErrWebhookSignatureInvalid
	}
	tsTime := time.Unix(tsNum, 0).UTC()
	if tsNum > 1e12 {
		tsTime = time.UnixMilli(tsNum).UTC()
	}
	if math.Abs(now.Sub(tsTime).Seconds()) > maxWebhookReplayAge.Seconds() {
		return ErrWebhookSignatureInvalid
	}
	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", strings.TrimSpace(dataID), strings.TrimSpace(requestID), tsStr)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	expected := mac.Sum(nil)
	for _, sig := range sigs {
		decoded, err := hex.DecodeString(sig)
		if err == nil && hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return ErrWebhookSignatureInvalid
}

func parseCommaKeyValues(header, tsKey, sigKey string) (string, []string) {
	var ts string
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case tsKey:
			if ts == "" {
				ts = v
			}
		case sigKey:
			if v != "" {
				sigs = append(sigs, v)
			}
		}
	}
	return ts, sigs
}
