package gist

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runGistCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewGistCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestGistListCmd(t *testing.T) {
	gists := []gitee.Gist{
		{ID: "abcdefgh12345678", Description: "My snippet", Public: false, UpdatedAt: time.Now(), Files: map[string]gitee.GistFile{"test.go": {}}},
	}
	out, err := runGistCmd([]string{"list"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gists)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "My snippet") {
		t.Errorf("expected description, got: %s", out)
	}
	if !strings.Contains(out, "secret") {
		t.Errorf("expected visibility, got: %s", out)
	}
}

func TestGistListCmd_shortID(t *testing.T) {
	gists := []gitee.Gist{
		{ID: "short", Description: "tiny id gist", UpdatedAt: time.Now(), Files: map[string]gitee.GistFile{}},
	}
	out, err := runGistCmd([]string{"list"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gists)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "short") {
		t.Errorf("expected short id, got: %s", out)
	}
}

func TestGistListCmd_empty(t *testing.T) {
	out, err := runGistCmd([]string{"list"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.Gist{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No gists found") && !strings.Contains(out, "未找到任何 Gist") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestGistListCmd_emptyJSON(t *testing.T) {
	out, err := runGistCmd([]string{"list", "--json"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.Gist{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out); got != "[]" {
		t.Fatalf("expected an empty JSON array, got %q", got)
	}
}

func TestGistViewCmd(t *testing.T) {
	g := gitee.Gist{
		ID:          "abc123",
		Description: "View me",
		Public:      true,
		Owner:       gitee.User{Login: "alice"},
		Files:       map[string]gitee.GistFile{"hello.go": {Content: "package main"}},
	}
	out, err := runGistCmd([]string{"view", "abc123"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(g)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "View me") {
		t.Errorf("expected description, got: %s", out)
	}
	if !strings.Contains(out, "package main") {
		t.Errorf("expected file content, got: %s", out)
	}
}

func TestGistDeleteCmd(t *testing.T) {
	called := false
	out, err := runGistCmd([]string{"delete", "abc123", "--yes"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			called = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected DELETE request")
	}
	if !strings.Contains(out, "Deleted gist abc123") && !strings.Contains(out, "已删除 Gist abc123") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestGistDeleteCmd_requiresYesInNonInteractiveMode(t *testing.T) {
	called := false
	_, err := runGistCmd([]string{"delete", "abc123"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected a --yes requirement, got %v", err)
	}
	if called {
		t.Fatal("delete request should not be sent without --yes")
	}
}
