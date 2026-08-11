package release

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func TestReleaseEditCmd_loadsAndSendsCompleteRelease(t *testing.T) {
	patchCalled := false
	out, err := runReleaseCmd([]string{
		"edit", "v1.2.0", "-R", "owner/repo",
		"--name", "Version 1.2 updated",
		"--body", "",
		"--prerelease=false",
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/releases/tags/v1.2.0":
			json.NewEncoder(w).Encode(gitee.Release{
				ID:         7,
				TagName:    "v1.2.0",
				Name:       "Version 1.2",
				Body:       "Old notes",
				Prerelease: true,
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/releases/7":
			patchCalled = true
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["tag_name"] != "v1.2.0" || payload["name"] != "Version 1.2 updated" {
				t.Errorf("expected hydrated tag and updated name, got %#v", payload)
			}
			if body, ok := payload["body"]; !ok || body != "" {
				t.Errorf("expected explicit empty body, got %#v", payload)
			}
			if prerelease, ok := payload["prerelease"]; !ok || prerelease != false {
				t.Errorf("expected explicit prerelease=false, got %#v", payload)
			}
			json.NewEncoder(w).Encode(gitee.Release{ID: 7, TagName: "v1.2.0", Name: "Version 1.2 updated"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !patchCalled {
		t.Fatal("expected PATCH request")
	}
	if !strings.Contains(out, "v1.2.0") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReleaseEditCmd_acceptsNumericID(t *testing.T) {
	_, err := runReleaseCmd([]string{"edit", "7", "-R", "owner/repo", "--tag", "v1.2.1"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != "/repos/owner/repo/releases/7" {
				t.Fatalf("unexpected GET path: %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(gitee.Release{ID: 7, TagName: "v1.2.0", Name: "Version 1.2"})
		case http.MethodPatch:
			json.NewEncoder(w).Encode(gitee.Release{ID: 7, TagName: "v1.2.1", Name: "Version 1.2"})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestReleaseEditCmd_requiresEditingFlagInNonInteractiveMode(t *testing.T) {
	called := false
	_, err := runReleaseCmd([]string{"edit", "7", "-R", "owner/repo"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected editing flag error, got %v", err)
	}
	if called {
		t.Fatal("API should not be called")
	}
}
