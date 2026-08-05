package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAuthenticatedUser(t *testing.T) {
	user := User{ID: 1, Login: "alice", Name: "Alice"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(user)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Login != "alice" {
		t.Errorf("expected alice, got %s", got.Login)
	}
}

func TestGetUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/bob" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(User{Login: "bob"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetUser(context.Background(), "bob")
	if err != nil {
		t.Fatal(err)
	}
	if got.Login != "bob" {
		t.Errorf("unexpected login: %s", got.Login)
	}
}

func TestSearchUsers(t *testing.T) {
	users := []User{{Login: "alice"}, {Login: "alex"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "al" {
			t.Errorf("expected q=al, got %s", r.URL.Query().Get("q"))
		}
		json.NewEncoder(w).Encode(users)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.SearchUsers(context.Background(), "al")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 users, got %d", len(got))
	}
}
