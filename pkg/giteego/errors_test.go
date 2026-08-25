package giteego

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestParseAPIErrorQuota(t *testing.T) {
	body := []byte(`{"message":"You exceeded your current quota, please check your plan and billing details.","type":"insufficient_quota","param":"","code":"insufficient_quota"}`)
	e := parseAPIError(http.StatusTooManyRequests, body)
	if e == nil {
		t.Fatal("expected non-nil error")
	}
	if !e.IsQuota() {
		t.Errorf("expected IsQuota, got %+v", e)
	}
	if e.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", e.StatusCode)
	}
	if !strings.Contains(e.Error(), "giteego: HTTP 429") {
		t.Errorf("expected stable error prefix, got %q", e.Error())
	}
	if !strings.Contains(e.Error(), "quota") {
		t.Errorf("expected quota hint in error, got %q", e.Error())
	}
}

func TestParseAPIErrorPlain429(t *testing.T) {
	// A bare rate-limit 429 without a quota body must NOT be labeled quota.
	e := parseAPIError(http.StatusTooManyRequests, []byte("rate limit"))
	if e == nil {
		t.Fatal("expected non-nil error")
	}
	if e.IsQuota() {
		t.Errorf("plain 429 should not be quota, got %+v", e)
	}
	if e.IsPermission() {
		t.Errorf("429 should not be permission, got %+v", e)
	}
}

func TestParseAPIError403(t *testing.T) {
	e := parseAPIError(http.StatusForbidden, []byte(`{"message":"forbidden"}`))
	if e == nil || !e.IsPermission() {
		t.Fatalf("expected permission error, got %+v", e)
	}
	if e.IsQuota() {
		t.Errorf("403 should not be quota, got %+v", e)
	}
}

func TestParseAPIErrorNonJSON(t *testing.T) {
	e := parseAPIError(http.StatusBadGateway, []byte("<html>bad gateway</html>"))
	if e == nil {
		t.Fatal("expected non-nil error")
	}
	if e.Message == "" {
		t.Errorf("expected raw body as message, got %+v", e)
	}
	if !strings.Contains(e.Error(), "giteego: HTTP 502") {
		t.Errorf("expected 502 prefix, got %q", e.Error())
	}
}

// TestDoSkipsRetryOnQuota asserts a 429 quota response is not retried: the
// handler runs exactly once and the returned error is an APIError.
func TestDoSkipsRetryOnQuota(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"quota exceeded","type":"insufficient_quota","code":"insufficient_quota"}`))
	}))
	defer srv.Close()

	c := NewClient("token", WithBaseURL(srv.URL), WithMaxRetries(3))
	req, err := c.newRequest(context.Background(), http.MethodGet, "/x", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.do(req, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Errorf("expected 1 request (no retry on quota), got %d", hits.Load())
	}
	var apiErr *APIError
	if !asAPIError(err, &apiErr) || !apiErr.IsQuota() {
		t.Errorf("expected quota APIError, got %T %v", err, err)
	}
}

// TestDoKeepsRetryingOn5xx asserts 5xx responses are still retried.
func TestDoKeepsRetryingOn5xx(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`oops`))
	}))
	defer srv.Close()

	c := NewClient("token", WithBaseURL(srv.URL), WithMaxRetries(2))
	req, err := c.newRequest(context.Background(), http.MethodGet, "/x", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.do(req, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 3 { // initial + 2 retries
		t.Errorf("expected 3 requests for 5xx, got %d", hits.Load())
	}
	var apiErr *APIError
	if !asAPIError(err, &apiErr) {
		t.Errorf("expected APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %+v", apiErr)
	}
}

func TestFetchAbsoluteURLSkipsRetryOnQuota(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"type":"insufficient_quota","code":"insufficient_quota"}`))
	}))
	defer srv.Close()

	c := NewClient("token", WithMaxRetries(3))
	_, err := c.FetchAbsoluteURL(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Errorf("expected 1 request, got %d", hits.Load())
	}
	var apiErr *APIError
	if !asAPIError(err, &apiErr) || !apiErr.IsQuota() {
		t.Errorf("expected quota APIError, got %T %v", err, err)
	}
}

// asAPIError mirrors errors.As without importing errors in this package's
// tests for clarity.
func asAPIError(err error, target **APIError) bool {
	for err != nil {
		if ae, ok := err.(*APIError); ok {
			*target = ae
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
