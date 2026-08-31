package giteego

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// roundTrip answers requests through the provided handler, recording requests
// for later assertions. It is the in-memory transport used across giteego tests.
type roundTrip struct {
	mu      sync.Mutex
	reqs    []*http.Request
	handler http.HandlerFunc
}

func (rt *roundTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.reqs = append(rt.reqs, r)
	rt.mu.Unlock()
	rec := httptest.NewRecorder()
	rt.handler(rec, r)
	return rec.Result(), nil
}

func (rt *roundTrip) requests() []*http.Request {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return append([]*http.Request(nil), rt.reqs...)
}

// testClient builds a Client wired to an in-memory transport (zero retries so
// tests stay deterministic unless WithMaxRetries is given).
func testClient(token string, handler http.HandlerFunc, opts ...Option) (*Client, *roundTrip) {
	rt := &roundTrip{handler: handler}
	base := []Option{
		WithHTTPClient(&http.Client{Transport: rt}),
		WithBaseURL("https://go-api.example/gitee-go"),
		WithMaxRetries(0),
	}
	base = append(base, opts...)
	return NewClient(token, base...), rt
}

func jsonBody(t *testing.T, data interface{}) string {
	t.Helper()
	b, err := json.Marshal(map[string]interface{}{"code": 0, "msg": "", "data": data})
	if err != nil {
		t.Fatalf("marshal json body: %v", err)
	}
	return string(b)
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("tok")
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.servicePath != "ipipe" {
		t.Errorf("servicePath = %q, want ipipe", c.servicePath)
	}
	if c.retries != defaultRetries {
		t.Errorf("retries = %d, want %d", c.retries, defaultRetries)
	}
	if c.accessToken != "tok" {
		t.Errorf("accessToken = %q, want tok", c.accessToken)
	}
}

func TestWithServicePathTrimsSlash(t *testing.T) {
	c := NewClient("tok", WithServicePath("/sa/"))
	if c.servicePath != "sa" {
		t.Errorf("servicePath = %q, want sa", c.servicePath)
	}
}

// TestNewRequestBuildsURL verifies baseURL + servicePath + path (+ query).
func TestNewRequestBuildsURL(t *testing.T) {
	c, _ := testClient("tok", nil)
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/pipelines", map[string]string{"ref": "master"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://go-api.example/gitee-go/ipipe/rest/v5/pipelines?ref=master"
	if req.URL.String() != want {
		t.Errorf("URL = %q, want %q", req.URL.String(), want)
	}
}

func TestNewRequestNoQuery(t *testing.T) {
	c, _ := testClient("tok", nil)
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/plugins", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://go-api.example/gitee-go/ipipe/rest/v5/plugins"
	if req.URL.String() != want {
		t.Errorf("URL = %q, want %q", req.URL.String(), want)
	}
}

func TestNewRequestHeaders(t *testing.T) {
	c, _ := testClient("tok", nil)
	req, err := c.newRequest(context.Background(), http.MethodPost, "/rest/v5/builds", nil, map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != "gitee-cli" {
		t.Errorf("User-Agent = %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer tok" {
		t.Errorf("Authorization = %q, want Bearer tok", got)
	}
	if req.Header.Get("Cookie") != "" {
		t.Errorf("Cookie should be empty when token set, got %q", req.Header.Get("Cookie"))
	}
	// Body is marshaled JSON.
	body, _ := io.ReadAll(req.Body)
	var sent map[string]string
	if err := json.Unmarshal(body, &sent); err != nil || sent["k"] != "v" {
		t.Errorf("body = %s, want JSON k=v", body)
	}
}

func TestNewRequestNoAuthNoCookie(t *testing.T) {
	c, _ := testClient("", nil)
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/pipelines", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" {
		t.Errorf("expected no auth headers, got Authorization=%q Cookie=%q",
			req.Header.Get("Authorization"), req.Header.Get("Cookie"))
	}
}

func TestNewRequestMarshalBodyError(t *testing.T) {
	c, _ := testClient("tok", nil)
	if _, err := c.newRequest(context.Background(), http.MethodPost, "/rest/v5/builds", nil, make(chan int)); err == nil ||
		!strings.Contains(err.Error(), "marshal body") {
		t.Errorf("expected marshal body error, got %v", err)
	}
}

func TestDoDecodesResultVO(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, jsonBody(t, map[string]interface{}{"id": 7}))
	})
	var out ResultVO[map[string]interface{}]
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/foo", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.do(req, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if out.Data["id"] != float64(7) {
		t.Errorf("Data = %v, want id 7", out.Data)
	}
}

func TestDoCodeNonZeroError(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"code":404,"msg":"pipeline not found","data":null}`)
	})
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/pipelines/1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.do(req, nil); err == nil || !strings.Contains(err.Error(), "pipeline not found") {
		t.Errorf("expected code!=0 error, got %v", err)
	}
}

func TestDoHTTPError(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "upstream timeout")
	})
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/pipelines", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.do(req, nil)
	if err == nil || !strings.Contains(err.Error(), "giteego: HTTP 502") {
		t.Errorf("expected HTTP error, got %v", err)
	}
}

func TestDoRetriesThenSucceeds(t *testing.T) {
	var mu sync.Mutex
	n := 0
	c, rt := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		attempt := n
		mu.Unlock()
		if attempt == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, "flaky")
			return
		}
		io.WriteString(w, jsonBody(t, map[string]interface{}{"id": 7}))
	}, WithMaxRetries(3))
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/foo", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out ResultVO[map[string]interface{}]
	if err := c.do(req, &out); err != nil {
		t.Fatalf("do after retry: %v", err)
	}
	mu.Lock()
	if n != 2 {
		mu.Unlock()
		t.Fatalf("expected 2 attempts, got %d", n)
	}
	mu.Unlock()
	if got := len(rt.requests()); got != 2 {
		t.Errorf("expected 2 recorded requests, got %d", got)
	}
}

func TestDoRetriesExhausted(t *testing.T) {
	var mu sync.Mutex
	n := 0
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, "still down")
	}, WithMaxRetries(2))
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/foo", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.do(req, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Errorf("expected HTTP 503 after retries, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if n != 3 {
		t.Errorf("expected 3 attempts (1 + 2 retries), got %d", n)
	}
}

func TestDoTransportErrorNoRetries(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	c.httpClient = &http.Client{Transport: errorTransport{err: errors.New("boom")}}
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/foo", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.do(req, nil)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected transport error, got %v", err)
	}
}

func TestFetchAbsoluteURLSuccess(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "https://go-api.example/logs/1" {
			t.Errorf("unexpected URL %q", r.URL.String())
		}
		io.WriteString(w, "log line 1\nlog line 2\n")
	})
	body, err := c.FetchAbsoluteURL(context.Background(), "https://go-api.example/logs/1")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "log line 1\nlog line 2\n" {
		t.Errorf("body = %q", body)
	}
}

func TestFetchAbsoluteURLError(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "forbidden")
	})
	_, err := c.FetchAbsoluteURL(context.Background(), "https://go-api.example/forbidden")
	if err == nil || !strings.Contains(err.Error(), "giteego: HTTP 403") {
		t.Errorf("expected HTTP 403, got %v", err)
	}
}

// TestMultiSourcePath verifies the multi-source route keeps its prefix under the
// ipipe service segment (used by the program pipeline family).
func TestMultiSourcePath(t *testing.T) {
	c, _ := testClient("tok", nil)
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/multi-source/pipelines", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://go-api.example/gitee-go/ipipe/rest/v5/multi-source/pipelines"
	if req.URL.String() != want {
		t.Errorf("URL = %q, want %q", req.URL.String(), want)
	}
}

// errorTransport always fails the round trip.
type errorTransport struct {
	err error
}

func (e errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, e.err
}

// TestDoLogsRequestURL verifies that the debug log includes the full request URL.
func TestDoLogsRequestURL(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, jsonBody(t, map[string]interface{}{"ok": true}))
	})
	req, err := c.newRequest(context.Background(), http.MethodGet, "/rest/v5/pipelines/builds/history", map[string]string{"ref": "master"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out ResultVO[map[string]interface{}]
	if err := c.do(req, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	logged := buf.String()
	if !strings.Contains(logged, "giteego HTTP request") {
		t.Errorf("expected debug log to contain 'giteego HTTP request', got: %s", logged)
	}
	if !strings.Contains(logged, "go-api.example/gitee-go/ipipe/rest/v5/pipelines/builds/history") {
		t.Errorf("expected debug log to contain the request URL, got: %s", logged)
	}
	if !strings.Contains(logged, "method=GET") {
		t.Errorf("expected debug log to contain method=GET, got: %s", logged)
	}
	if !strings.Contains(logged, "giteego HTTP response") {
		t.Errorf("expected debug log to contain 'giteego HTTP response', got: %s", logged)
	}
}
