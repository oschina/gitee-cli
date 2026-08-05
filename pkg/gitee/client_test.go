package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/build"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("test-token", WithBaseURL(srv.URL))
	return c, srv
}

func TestNewClient_defaults(t *testing.T) {
	c := NewClient("my-token")
	if c.accessToken != "my-token" {
		t.Fatalf("expected token my-token, got %s", c.accessToken)
	}
	if c.baseURL != defaultBaseURL {
		t.Fatalf("expected default base URL, got %s", c.baseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("tok", WithBaseURL("https://example.com"))
	if c.baseURL != "https://example.com" {
		t.Fatalf("expected custom base URL")
	}
}

func TestDo_successJSON(t *testing.T) {
	type payload struct {
		Login string `json:"login"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload{Login: "alice"})
	}))
	defer srv.Close()

	c := NewClient("test-token", WithBaseURL(srv.URL))
	req, _ := c.newRequest(context.Background(), http.MethodGet, "/user", nil, nil)
	var result payload
	if err := c.do(req, &result); err != nil {
		t.Fatal(err)
	}
	if result.Login != "alice" {
		t.Fatalf("expected alice, got %s", result.Login)
	}
}

func TestDo_errorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	req, _ := c.newRequest(context.Background(), http.MethodGet, "/missing", nil, nil)
	err := c.do(req, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	apiErr, ok := err.(*ErrorResponse)
	if !ok {
		t.Fatalf("expected *ErrorResponse, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestDo_noContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	req, _ := c.newRequest(context.Background(), http.MethodDelete, "/res/1", nil, nil)
	if err := c.do(req, nil); err != nil {
		t.Fatal(err)
	}
}

func TestNewRequest_queryParams(t *testing.T) {
	c := NewClient("tok", WithBaseURL("https://api.example.com"))
	req, err := c.newRequest(context.Background(), http.MethodGet, "/items", map[string]string{"page": "2", "per_page": "10"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	if q.Get("page") != "2" {
		t.Errorf("expected page=2, got %s", q.Get("page"))
	}
	if q.Get("per_page") != "10" {
		t.Errorf("expected per_page=10, got %s", q.Get("per_page"))
	}
}

func TestNewRequest_jsonBody(t *testing.T) {
	c := NewClient("tok", WithBaseURL("https://api.example.com"))
	body := map[string]string{"title": "hello"}
	req, err := c.newRequest(context.Background(), http.MethodPost, "/items", nil, body)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json content type")
	}
	if got := req.Header.Get("User-Agent"); got != build.UserAgent() {
		t.Errorf("expected %q user agent, got %q", build.UserAgent(), got)
	}
}

func TestErrorResponse_Error(t *testing.T) {
	e := &ErrorResponse{StatusCode: 422, Message: "validation failed"}
	got := e.Error()
	want := "gitee: HTTP 422: validation failed"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRawDoDoesNotAuthorizeCrossOrigin(t *testing.T) {
	var gotAuthorization string
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer external.Close()

	c := NewClient("secret-token", WithBaseURL("https://gitee.example.com/api/v5"))
	req, err := http.NewRequest(http.MethodGet, external.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotAuthorization != "" {
		t.Fatalf("cross-origin request leaked Authorization: %q", gotAuthorization)
	}
}

func TestRawDoAuthorizesConfiguredOrigin(t *testing.T) {
	var gotAuthorization, gotUserAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("secret-token", WithBaseURL(srv.URL+"/api/v5"))
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/user", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotAuthorization != "Bearer secret-token" {
		t.Fatalf("expected configured origin to receive token, got %q", gotAuthorization)
	}
	if gotUserAgent != build.UserAgent() {
		t.Fatalf("expected Gitee CLI user agent, got %q", gotUserAgent)
	}
}

func TestRawDoPreservesCustomUserAgent(t *testing.T) {
	var gotUserAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("secret-token", WithBaseURL(srv.URL))
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/user", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "custom-client/1.0")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotUserAgent != "custom-client/1.0" {
		t.Fatalf("expected custom user agent to be preserved, got %q", gotUserAgent)
	}
}

func TestRedirectStripsAuthorizationAcrossOrigins(t *testing.T) {
	var redirectedAuthorization string
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer external.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("origin did not receive Authorization: %q", got)
		}
		http.Redirect(w, r, external.URL, http.StatusFound)
	}))
	defer origin.Close()

	c := NewClient("secret-token", WithBaseURL(origin.URL))
	req, err := http.NewRequest(http.MethodGet, origin.URL+"/redirect", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if redirectedAuthorization != "" {
		t.Fatalf("redirect leaked Authorization across origins: %q", redirectedAuthorization)
	}
}
