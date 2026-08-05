package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestShouldRetryStatus(t *testing.T) {
	cfg := RetryConfig{RetryOn429: true, RetryOn5xx: true}

	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{201, false},
		{301, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{422, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
	}
	for _, tt := range tests {
		got := shouldRetryStatus(tt.code, cfg)
		if got != tt.want {
			t.Errorf("shouldRetryStatus(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestShouldRetryStatus_disabled(t *testing.T) {
	cfg429Off := RetryConfig{RetryOn429: false, RetryOn5xx: true}
	if shouldRetryStatus(429, cfg429Off) {
		t.Error("429 should not be retried when RetryOn429=false")
	}

	cfg5xxOff := RetryConfig{RetryOn429: true, RetryOn5xx: false}
	if shouldRetryStatus(500, cfg5xxOff) {
		t.Error("500 should not be retried when RetryOn5xx=false")
	}
}

func TestCalculateBackoff_exponential(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
	}

	prev := time.Duration(0)
	for attempt := 0; attempt < 5; attempt++ {
		d := calculateBackoff(attempt, cfg)
		base := cfg.InitialDelay << attempt
		if d < base || d > base*3/2 {
			t.Errorf("attempt %d: backoff %v outside expected range [%v, %v]", attempt, d, base, base*3/2)
		}
		if attempt > 0 && d < prev/2 {
			t.Errorf("attempt %d: backoff %v should be >= previous base", attempt, d)
		}
		prev = base
	}
}

func TestCalculateBackoff_capAtMax(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
	}

	for attempt := 10; attempt < 15; attempt++ {
		d := calculateBackoff(attempt, cfg)
		if d > cfg.MaxDelay*3/2 {
			t.Errorf("attempt %d: backoff %v exceeds max*1.5 (%v)", attempt, d, cfg.MaxDelay*3/2)
		}
	}
}

func TestParseRetryAfter_seconds(t *testing.T) {
	d := parseRetryAfter("5")
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}

func TestParseRetryAfter_httpDate(t *testing.T) {
	future := time.Now().Add(10 * time.Second).UTC()
	header := future.Format(http.TimeFormat)
	d := parseRetryAfter(header)
	if d < 5*time.Second || d > 15*time.Second {
		t.Errorf("expected ~10s, got %v", d)
	}
}

func TestParseRetryAfter_empty(t *testing.T) {
	d := parseRetryAfter("")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_invalid(t *testing.T) {
	d := parseRetryAfter("not-a-number-or-date")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func testRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		RetryOn429:   true,
		RetryOn5xx:   true,
	}
}

func TestRetry_on429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"login": "alice"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL), WithRetryConfig(testRetryConfig()))
	req, _ := c.newRequest(context.Background(), http.MethodGet, "/user", nil, nil)
	var result struct {
		Login string `json:"login"`
	}
	if err := c.do(req, &result); err != nil {
		t.Fatal(err)
	}
	if result.Login != "alice" {
		t.Errorf("expected alice, got %s", result.Login)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected 3 calls, got %d", got)
	}
}

func TestRetry_on5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL), WithRetryConfig(testRetryConfig()))
	req, _ := c.newRequest(context.Background(), http.MethodGet, "/health", nil, nil)
	var result struct {
		Status string `json:"status"`
	}
	if err := c.do(req, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Errorf("expected ok, got %s", result.Status)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 calls, got %d", got)
	}
}

func TestRetry_noRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL), WithRetryConfig(testRetryConfig()))
	req, _ := c.newRequest(context.Background(), http.MethodGet, "/missing", nil, nil)
	err := c.do(req, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 call (no retry), got %d", got)
	}
}

func TestRetry_exhaustedRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"message": "unavailable"})
	}))
	defer srv.Close()

	cfg := testRetryConfig()
	cfg.MaxRetries = 2
	c := NewClient("tok", WithBaseURL(srv.URL), WithRetryConfig(cfg))
	req, _ := c.newRequest(context.Background(), http.MethodGet, "/down", nil, nil)
	err := c.do(req, nil)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	apiErr, ok := err.(*ErrorResponse)
	if !ok {
		t.Fatalf("expected *ErrorResponse, got %T", err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", apiErr.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", got)
	}
}

func TestRetry_contextCancellation(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := RetryConfig{
		MaxRetries:   10,
		InitialDelay: 5 * time.Second,
		MaxDelay:     10 * time.Second,
		RetryOn5xx:   true,
	}
	c := NewClient("tok", WithBaseURL(srv.URL), WithRetryConfig(cfg))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, _ := c.newRequest(ctx, http.MethodGet, "/slow", nil, nil)
	err := c.do(req, nil)
	if err == nil {
		t.Fatal("expected error on context cancellation")
	}
	if got := atomic.LoadInt32(&calls); got > 2 {
		t.Errorf("expected at most 2 calls before cancel, got %d", got)
	}
}

func TestWithMaxRetries(t *testing.T) {
	c := NewClient("tok", WithMaxRetries(5))
	if c.retryConfig.MaxRetries != 5 {
		t.Errorf("expected MaxRetries=5, got %d", c.retryConfig.MaxRetries)
	}
}

func TestRetry_respectsRetryAfterHeader(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	cfg := testRetryConfig()
	c := NewClient("tok", WithBaseURL(srv.URL), WithRetryConfig(cfg))
	req, _ := c.newRequest(context.Background(), http.MethodGet, "/rate", nil, nil)
	var result map[string]string
	if err := c.do(req, &result); err != nil {
		t.Fatal(err)
	}
	if result["ok"] != "true" {
		t.Errorf("expected ok=true, got %v", result)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 calls, got %d", got)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	c := NewClient("tok")
	if c.retryConfig.MaxRetries != 3 {
		t.Errorf("expected default MaxRetries=3, got %d", c.retryConfig.MaxRetries)
	}
	if c.retryConfig.InitialDelay != 1*time.Second {
		t.Errorf("expected default InitialDelay=1s, got %v", c.retryConfig.InitialDelay)
	}
	if c.retryConfig.MaxDelay != 30*time.Second {
		t.Errorf("expected default MaxDelay=30s, got %v", c.retryConfig.MaxDelay)
	}
	if !c.retryConfig.RetryOn429 {
		t.Error("expected default RetryOn429=true")
	}
	if !c.retryConfig.RetryOn5xx {
		t.Error("expected default RetryOn5xx=true")
	}
}
