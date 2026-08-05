package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSSHKeys(t *testing.T) {
	keys := []SSHKey{{ID: 1, Key: "ssh-rsa AAA"}, {ID: 2, Key: "ssh-ed25519 BBB"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/keys" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(keys)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ListSSHKeys(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(got))
	}
	if got[0].ID != 1 {
		t.Errorf("unexpected key id: %d", got[0].ID)
	}
}

func TestDeleteSSHKey(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/user/keys/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	if err := c.DeleteSSHKey(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler not called")
	}
}
