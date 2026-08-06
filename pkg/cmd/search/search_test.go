package search

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runSearchCmd(t *testing.T, args []string, handler http.Handler) (string, error) {
	t.Helper()
	factory := cmdtest.NewTestFactory(handler)
	defer factory.Close()
	cmd := NewSearchCmd(factory.Factory)
	cmd.SetOut(factory.OutBuf)
	cmd.SetErr(factory.ErrOutBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return factory.Output(), err
}

func TestReposCommand(t *testing.T) {
	var query map[string]string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = firstQueryValues(r)
		json.NewEncoder(w).Encode([]gitee.Repository{{
			FullName: "oschina/gitee-cli", Description: "Gitee CLI", Language: "Go", StargazersCount: 42,
		}})
	})
	out, err := runSearchCmd(t, []string{
		"repos", "cli", "--owner", "oschina", "--fork", "--language", "Go",
		"--sort", "stars_count", "--order", "asc", "--page", "2", "--limit", "50",
	}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "oschina/gitee-cli") || !strings.Contains(out, "Gitee CLI") {
		t.Fatalf("unexpected output: %s", out)
	}
	want := map[string]string{
		"q": "cli", "owner": "oschina", "fork": "true", "language": "Go",
		"sort": "stars_count", "order": "asc", "page": "2", "per_page": "50",
	}
	assertValues(t, query, want)
}

func TestIssuesCommand(t *testing.T) {
	var query map[string]string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = firstQueryValues(r)
		json.NewEncoder(w).Encode([]gitee.Issue{{
			Number: "I123", Title: "Request timeout", State: "open",
			User: gitee.User{Login: "alice"}, Repository: gitee.Repository{FullName: "oschina/gitee"},
		}})
	})
	out, err := runSearchCmd(t, []string{
		"issues", "timeout", "--repo", "oschina/gitee", "--label", "bug", "--state", "open",
		"--author", "alice", "--assignee", "bob", "--language", "Ruby", "--sort", "updated_at",
	}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "oschina/gitee") || !strings.Contains(out, "Request timeout") {
		t.Fatalf("unexpected output: %s", out)
	}
	want := map[string]string{
		"q": "timeout", "repo": "oschina/gitee", "label": "bug", "state": "open",
		"author": "alice", "assignee": "bob", "language": "Ruby", "sort": "updated_at",
	}
	assertValues(t, query, want)
}

func TestUsersCommandJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.User{{Login: "alice", Name: "Alice"}})
	})
	out, err := runSearchCmd(t, []string{"users", "alice", "--json=login,name"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"login":"alice"`) || !strings.Contains(out, `"name":"Alice"`) {
		t.Fatalf("unexpected JSON output: %s", out)
	}
}

func TestSearchValidation(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"repos", "cli", "--limit", "101"}, "--limit must be between 1 and 100"},
		{[]string{"issues", "bug", "--state", "unknown"}, "--state must be one of"},
		{[]string{"users", "alice", "--sort", "stars_count"}, "--sort must be one of"},
	}
	for _, tt := range tests {
		_, err := runSearchCmd(t, tt.args, http.NotFoundHandler())
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("args %v: error = %v, want containing %q", tt.args, err, tt.want)
		}
	}
}

func TestSearchRequiresSubcommandAndQuery(t *testing.T) {
	if _, err := runSearchCmd(t, []string{"repos"}, http.NotFoundHandler()); err == nil {
		t.Fatal("expected missing query error")
	}
}

func firstQueryValues(r *http.Request) map[string]string {
	values := make(map[string]string)
	for key, entries := range r.URL.Query() {
		if len(entries) > 0 {
			values[key] = entries[0]
		}
	}
	return values
}

func assertValues(t *testing.T, got, want map[string]string) {
	t.Helper()
	for key, value := range want {
		if got[key] != value {
			t.Errorf("query %s = %q, want %q", key, got[key], value)
		}
	}
}
