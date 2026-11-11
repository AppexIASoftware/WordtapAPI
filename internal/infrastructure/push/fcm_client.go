package push

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type PushSender interface {
	Send(ctx context.Context, token, title, body string, data map[string]string) error
}

type FCMServiceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

type FCMClient struct {
	projectID   string
	clientEmail string
	privateKey  *rsa.PrivateKey
	httpClient  *http.Client

	tokenMu     sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

func NewFCMClientFromCredentials(credentialsJSON []byte) (*FCMClient, error) {
	var sa FCMServiceAccount
	if err := json.Unmarshal(credentialsJSON, &sa); err != nil {
		return nil, fmt.Errorf("invalid firebase credentials json: %w", err)
	}

	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, fmt.Errorf("missing required fields in firebase credentials")
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(sa.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rsa private key: %w", err)
	}

	return &FCMClient{
		projectID:   sa.ProjectID,
		clientEmail: sa.ClientEmail,
		privateKey:  key,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func NewFCMClient() (*FCMClient, error) {
	// 1. Env variable with raw JSON
	if raw := strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_JSON")); raw != "" {
		return NewFCMClientFromCredentials([]byte(raw))
	}

	// 2. Explicit path in env
	if path := strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_PATH")); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read firebase credentials from path %s: %w", path, err)
		}
		return NewFCMClientFromCredentials(data)
	}

	// 3. Fallback auto-discovery in working directory
	matches, _ := filepath.Glob("*firebase-adminsdk*.json")
	if len(matches) > 0 {
		data, err := os.ReadFile(matches[0])
		if err == nil {
			return NewFCMClientFromCredentials(data)
		}
	}

	return nil, fmt.Errorf("no firebase credentials found")
}

func (c *FCMClient) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-5*time.Minute)) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check under lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-5*time.Minute)) {
		return c.accessToken, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   c.clientEmail,
		"sub":   c.clientEmail,
		"aud":   "https://oauth2.googleapis.com/token",
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedJWT, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign oauth jwt assertion: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", signedJWT)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token with google: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google token exchange failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse google token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return c.accessToken, nil
}

func (c *FCMClient) Send(ctx context.Context, token, title, body string, data map[string]string) error {
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("cannot send fcm push: %w", err)
	}

	message := map[string]any{
		"token": token,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
	}
	if len(data) > 0 {
		message["data"] = data
	}

	payload := map[string]any{
		"message": message,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", c.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fcm send request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fcm send failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

// NoOpPushSender is a fallback sender when no push credentials are provided.
type NoOpPushSender struct{}

func (n *NoOpPushSender) Send(_ context.Context, _, _, _ string, _ map[string]string) error {
	return nil
}
