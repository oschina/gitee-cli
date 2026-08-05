package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/build"
)

func TestCompleteSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != build.UserAgent() {
			t.Errorf("expected %q user agent, got %q", build.UserAgent(), got)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{{
				"message": map[string]string{"role": "assistant", "content": "done"},
			}},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token", "model")
	got, err := client.Complete(context.Background(), "", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if got != "done" {
		t.Fatalf("expected response %q, got %q", "done", got)
	}
}

func TestCompleteStreamSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != build.UserAgent() {
			t.Errorf("expected %q user agent, got %q", build.UserAgent(), got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"done"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token", "model")
	got, err := client.CompleteStream(context.Background(), []Message{{Role: "user", Content: "hello"}}, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if got != "done" {
		t.Fatalf("expected response %q, got %q", "done", got)
	}
}
