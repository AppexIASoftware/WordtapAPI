package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidGoogleToken = errors.New("invalid or expired google id token")
	ErrGoogleEmailMissing = errors.New("google token missing email")
)

type GoogleTokenClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Audience      string `json:"aud"`
}

type GoogleVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*GoogleTokenClaims, error)
}

type HTTPGoogleVerifier struct {
	httpClient *http.Client
	clientID   string
}

func NewHTTPGoogleVerifier(clientID string) *HTTPGoogleVerifier {
	return &HTTPGoogleVerifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		clientID:   strings.TrimSpace(clientID),
	}
}

// rawGoogleTokeninfo mapea la respuesta de https://oauth2.googleapis.com/tokeninfo
type rawGoogleTokeninfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"` // Puede ser string "true" o bool true
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Aud           string `json:"aud"`
	Error         string `json:"error"`
	ErrorDesc     string `json:"error_description"`
}

func (v *HTTPGoogleVerifier) VerifyIDToken(ctx context.Context, idToken string) (*GoogleTokenClaims, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, ErrInvalidGoogleToken
	}

	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create google verification request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google verification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidGoogleToken
	}

	var raw rawGoogleTokeninfo
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to parse google token response: %w", err)
	}

	if raw.Sub == "" || raw.Email == "" {
		return nil, ErrGoogleEmailMissing
	}

	// Validar audiencia si se configuró GOOGLE_CLIENT_ID
	if v.clientID != "" && raw.Aud != v.clientID {
		return nil, fmt.Errorf("token audience mismatch: expected %s, got %s", v.clientID, raw.Aud)
	}

	verified := false
	switch val := raw.EmailVerified.(type) {
	case bool:
		verified = val
	case string:
		b, _ := strconv.ParseBool(val)
		verified = b
	}

	return &GoogleTokenClaims{
		Sub:           raw.Sub,
		Email:         strings.ToLower(strings.TrimSpace(raw.Email)),
		EmailVerified: verified,
		Name:          raw.Name,
		Picture:       raw.Picture,
		Audience:      raw.Aud,
	}, nil
}
