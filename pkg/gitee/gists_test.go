package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListGists(t *testing.T) {
	gists := []Gist{{ID: "abc123", Description: "test gist"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gists" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(gists)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ListGists(context.Background(), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 gist, got %d", len(got))
	}
	if got[0].ID != "abc123" {
		t.Errorf("unexpected gist id: %s", got[0].ID)
	}
}

func TestGetGist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gists/abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Gist{ID: "abc123", Description: "hello"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetGist(context.Background(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "hello" {
		t.Errorf("unexpected description: %s", got.Description)
	}
}

func TestCreateGist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var p CreateGistParams
		json.NewDecoder(r.Body).Decode(&p)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Gist{ID: "newid", Description: p.Description, HTMLURL: "https://gitee.com/gist/newid"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.CreateGist(context.Background(), &CreateGistParams{
		Description: "my gist",
		Files:       map[string]GistFile{"test.txt": {Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "newid" {
		t.Errorf("unexpected id: %s", got.ID)
	}
}

func TestDeleteGist(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	if err := c.DeleteGist(context.Background(), "abc123"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler not called")
	}
}
