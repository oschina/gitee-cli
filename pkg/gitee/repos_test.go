package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRepo(t *testing.T) {
	repo := Repository{FullName: "owner/repo", Description: "test repo"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(repo)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetRepo(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if got.FullName != "owner/repo" {
		t.Errorf("unexpected repo: %+v", got)
	}
}

func TestListUserRepos(t *testing.T) {
	repos := []Repository{
		{FullName: "alice/a"},
		{FullName: "alice/b"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repos" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ListUserRepos(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(got))
	}
}

func TestCreateRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var p CreateRepoParams
		json.NewDecoder(r.Body).Decode(&p)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Repository{FullName: "owner/" + p.Name, Name: p.Name})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.CreateRepo(context.Background(), &CreateRepoParams{Name: "new-repo"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "new-repo" {
		t.Errorf("unexpected repo name: %s", got.Name)
	}
}

func TestDeleteRepo(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	if err := c.DeleteRepo(context.Background(), "owner", "repo"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler not called")
	}
}

func TestListUserRepos_withParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sort") != "updated" {
			t.Errorf("expected sort=updated")
		}
		json.NewEncoder(w).Encode([]Repository{})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	_, err := c.ListUserRepos(context.Background(), &ListReposParams{Sort: "updated"})
	if err != nil {
		t.Fatal(err)
	}
}
