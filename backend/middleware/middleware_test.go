package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options: nosniff")
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options: DENY")
	}
	if resp.Header.Get("X-XSS-Protection") != "0" {
		t.Error("expected X-XSS-Protection: 0")
	}
	if resp.Header.Get("Referrer-Policy") == "" {
		t.Error("expected Referrer-Policy header")
	}
}

func TestRateLimiterUnderLimit(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	handler := rl.Limit(okHandler())

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Result().StatusCode)
		}
	}
}

func TestRateLimiterExceeded(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	handler := rl.Limit(okHandler())

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Result().StatusCode)
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter(1, 50*time.Millisecond)
	handler := rl.Limit(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("expected first request to succeed")
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatal("expected second request to be rate limited")
	}

	time.Sleep(60 * time.Millisecond)

	req3 := httptest.NewRequest("GET", "/", nil)
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if w3.Result().StatusCode != http.StatusOK {
		t.Fatal("expected request after window to succeed")
	}
}

func TestRateLimitHeaders(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	handler := rl.Limit(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.Header.Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
	if resp.Header.Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

func TestLoggerRecordsStatus(t *testing.T) {
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Result().StatusCode)
	}
}
