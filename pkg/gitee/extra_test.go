package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatePull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var p CreatePullParams
		json.NewDecoder(r.Body).Decode(&p)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PullRequest{Number: 99, Title: p.Title})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.CreatePull(context.Background(), "owner", "repo", &CreatePullParams{
		Title: "New PR",
		Head:  "feature",
		Base:  "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "New PR" {
		t.Errorf("unexpected title: %s", got.Title)
	}
}

func TestCreatePullComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/pulls/5/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		json.NewEncoder(w).Encode(Comment{ID: 1, Body: payload["body"]})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.CreatePullComment(context.Background(), "owner", "repo", 5, "LGTM")
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "LGTM" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestGetPullDiff(t *testing.T) {
	diffContent := "diff --git a/file.go b/file.go\n+added line"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls/3.diff" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(diffContent))
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetPullDiff(context.Background(), "owner", "repo", 3)
	if err != nil {
		t.Fatal(err)
	}
	if got != diffContent {
		t.Errorf("unexpected diff:\ngot:  %q\nwant: %q", got, diffContent)
	}
}

func TestCreateIssueComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/issues/IJEE1/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		json.NewEncoder(w).Encode(Comment{ID: 2, Body: payload["body"]})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.CreateIssueComment(context.Background(), "owner", "repo", "IJEE1", "Nice work!")
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "Nice work!" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestGetLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Release{ID: 99, TagName: "v9.9.9"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetLatestRelease(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if got.TagName != "v9.9.9" {
		t.Errorf("unexpected tag: %s", got.TagName)
	}
}

func TestListOwnerRepos(t *testing.T) {
	repos := []Repository{{FullName: "org/a"}, {FullName: "org/b"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/org/repos" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ListOwnerRepos(context.Background(), "org", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(got))
	}
}

func TestForkRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/upstream/repo/forks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(Repository{FullName: "me/repo"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ForkRepo(context.Background(), "upstream", "repo", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.FullName != "me/repo" {
		t.Errorf("unexpected fork: %s", got.FullName)
	}
}
