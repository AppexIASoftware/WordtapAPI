package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestBodyLimitMiddleware(t *testing.T) {
	e := echo.New()
	e.POST("/limited", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, BodyLimit(64))

	okReq := httptest.NewRequest(http.MethodPost, "/limited", bytes.NewReader(make([]byte, 16)))
	okRec := httptest.NewRecorder()
	e.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for 16-byte payload, got %d", okRec.Code)
	}

	oversizeReq := httptest.NewRequest(http.MethodPost, "/limited", bytes.NewReader(make([]byte, 128)))
	oversizeRec := httptest.NewRecorder()
	e.ServeHTTP(oversizeRec, oversizeReq)
	if oversizeRec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 Request Entity Too Large for 128-byte payload, got %d", oversizeRec.Code)
	}
}

func TestIPRateLimiterMiddleware(t *testing.T) {
	e := echo.New()
	e.GET("/ping", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, IPRateLimiter(1, 2))

	doReq := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = ip + ":12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := doReq("198.51.100.10"); code != http.StatusOK {
		t.Fatalf("1st request: expected 200, got %d", code)
	}
	if code := doReq("198.51.100.10"); code != http.StatusOK {
		t.Fatalf("2nd request: expected 200, got %d", code)
	}
	if code := doReq("198.51.100.10"); code != http.StatusTooManyRequests {
		t.Fatalf("3rd request exceeding burst: expected 429, got %d", code)
	}
	if code := doReq("198.51.100.20"); code != http.StatusOK {
		t.Fatalf("distinct IP should not share bucket: expected 200, got %d", code)
	}
}
