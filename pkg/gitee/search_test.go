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
		if r.URL.Path != "/search/repositories" {
			t.Fatalf("path = %q, want /search/repositories", r.URL.Path)
		}
		want := map[string]string{
			"q": "cli", "owner": "oschina", "fork": "true", "language": "Go",
			"sort": "stars_count", "order": "asc", "page": "2", "per_page": "50",
		}
		assertQuery(t, r, want)
		json.NewEncoder(w).Encode([]Repository{{FullName: "oschina/gitee-cli"}})
	}))
	defer server.Close()

	client := NewClient("token", WithBaseURL(server.URL))
	repos, err := client.SearchRepositories(context.Background(), &SearchRepositoriesParams{
		Query: "cli", Owner: "oschina", Fork: true, Language: "Go",
		Sort: "stars_count", Order: "asc", Page: 2, PerPage: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].FullName != "oschina/gitee-cli" {
		t.Fatalf("unexpected repositories: %#v", repos)
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
