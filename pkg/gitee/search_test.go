package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/repos" {
			t.Fatalf("path = %q, want /search/repos", r.URL.Path)
		}
		want := map[string]string{
			"q": "cli", "language": "Go", "size": "50", "from": "50",
		}
		assertQuery(t, r, want)
		for _, key := range []string{"owner", "fork", "sort", "order", "page", "per_page"} {
			if got := r.URL.Query().Get(key); got != "" {
				t.Errorf("query %s = %q, want empty", key, got)
			}
		}
		json.NewEncoder(w).Encode(searchRepoResult{
			TotalCount: 1,
			Data: []searchRepoItem{{
				ID: 1, Name: "gitee-cli", Title: "oschina/gitee-cli",
				URL: "https://gitee.com/oschina/gitee-cli", Owner: "oschina", Path: "gitee-cli",
				Description: "Gitee CLI", Stars: 42, Forks: 5, License: "MIT",
				Languages: []string{"Go"}, UpdatedAt: "2026-09-24 10:00:00",
			}},
		})
	}))
	defer server.Close()

	client := NewClient("token", WithBaseURL(server.URL))
	repos, err := client.SearchRepositories(context.Background(), &SearchRepositoriesParams{
		Query: "cli", Language: "Go", Page: 2, PerPage: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("unexpected repository count: %d", len(repos))
	}
	repo := repos[0]
	if repo.FullName != "oschina/gitee-cli" {
		t.Errorf("FullName = %q, want oschina/gitee-cli", repo.FullName)
	}
	if repo.HTMLURL != "https://gitee.com/oschina/gitee-cli" {
		t.Errorf("HTMLURL = %q, want https://gitee.com/oschina/gitee-cli", repo.HTMLURL)
	}
	if repo.StargazersCount != 42 {
		t.Errorf("StargazersCount = %d, want 42", repo.StargazersCount)
	}
	if repo.Language != "Go" {
		t.Errorf("Language = %q, want Go", repo.Language)
	}
	if repo.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestSearchIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/issues" {
			t.Fatalf("path = %q, want /search/issues", r.URL.Path)
		}
		want := map[string]string{
			"q": "timeout", "repo": "oschina/gitee", "language": "Ruby", "label": "bug",
			"state": "open", "author": "alice", "assignee": "bob",
			"sort": "updated_at", "order": "desc", "page": "3", "per_page": "25",
		}
		assertQuery(t, r, want)
		json.NewEncoder(w).Encode([]Issue{{Number: "I1", Repository: Repository{FullName: "oschina/gitee"}}})
	}))
	defer server.Close()

	client := NewClient("token", WithBaseURL(server.URL))
	issues, err := client.SearchIssues(context.Background(), &SearchIssuesParams{
		Query: "timeout", Repo: "oschina/gitee", Language: "Ruby", Label: "bug",
		State: "open", Author: "alice", Assignee: "bob", Sort: "updated_at",
		Order: "desc", Page: 3, PerPage: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Repository.FullName != "oschina/gitee" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestSearchUsersWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/users" {
			t.Fatalf("path = %q, want /search/users", r.URL.Path)
		}
		assertQuery(t, r, map[string]string{
			"q": "alice", "sort": "joined_at", "order": "desc", "page": "4", "per_page": "10",
		})
		json.NewEncoder(w).Encode([]User{{Login: "alice"}})
	}))
	defer server.Close()

	client := NewClient("token", WithBaseURL(server.URL))
	users, err := client.SearchUsersWithParams(context.Background(), &SearchUsersParams{
		Query: "alice", Sort: "joined_at", Order: "desc", Page: 4, PerPage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Login != "alice" {
		t.Fatalf("unexpected users: %#v", users)
	}
}

func assertQuery(t *testing.T, r *http.Request, want map[string]string) {
	t.Helper()
	query := r.URL.Query()
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("query %s = %q, want %q", key, got, value)
		}
	}
}
