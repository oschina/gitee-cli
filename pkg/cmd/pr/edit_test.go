package pr

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func TestPREditCmd_sendsExplicitEmptyBodyAndFalseDraft(t *testing.T) {
	called := false
	out, err := runPRCmd([]string{
		"edit", "42", "-R", "owner/repo",
		"--title", "Updated title",
		"--body", "",
		"--draft=false",
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/repos/owner/repo/pulls/42" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["title"] != "Updated title" {
			t.Errorf("unexpected title: %#v", payload["title"])
		}
		if body, ok := payload["body"]; !ok || body != "" {
			t.Errorf("expected explicit empty body, got %#v", payload)
		}
		if draft, ok := payload["draft"]; !ok || draft != false {
			t.Errorf("expected explicit draft=false, got %#v", payload)
		}
		json.NewEncoder(w).Encode(gitee.PullRequest{
			Number:  42,
			Title:   "Updated title",
			HTMLURL: "https://gitee.com/owner/repo/pulls/42",
		})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected PATCH request")
	}
	if !strings.Contains(out, "#42") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestPREditCmd_requiresEditingFlagInNonInteractiveMode(t *testing.T) {
	called := false
	_, err := runPRCmd([]string{"edit", "42", "-R", "owner/repo"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected editing flag error, got %v", err)
	}
	if called {
		t.Fatal("API should not be called")
	}
}

func TestPREditCmd_rejectsEmptyTitle(t *testing.T) {
	_, err := runPRCmd([]string{"edit", "42", "-R", "owner/repo", "--title", ""}, http.NotFoundHandler())
	if err == nil || !strings.Contains(err.Error(), "title cannot be empty") {
		t.Fatalf("expected empty title error, got %v", err)
	}
}
