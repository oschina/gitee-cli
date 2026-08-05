package repo

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runRepoCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewRepoCmd(tf.Factory)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestRepoListCmd(t *testing.T) {
	repos := []gitee.Repository{
		{FullName: "alice/project-a", Description: "First project"},
		{FullName: "alice/project-b"},
	}
	out, err := runRepoCmd([]string{"list"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(repos)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice/project-a") {
		t.Errorf("expected repo name, got: %s", out)
	}
}

func TestRepoListCmd_json(t *testing.T) {
	repos := []gitee.Repository{{FullName: "alice/repo"}}
	out, err := runRepoCmd([]string{"list", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(repos)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice/repo") {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestRepoViewCmd(t *testing.T) {
	repo := gitee.Repository{
		FullName:    "alice/myrepo",
		Description: "My repository",
		HTMLURL:     "https://gitee.com/alice/myrepo",
	}
	out, err := runRepoCmd([]string{"view", "alice/myrepo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(repo)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice/myrepo") {
		t.Errorf("expected repo name, got: %s", out)
	}
}

func TestRepoDeleteCmd(t *testing.T) {
	called := false
	out, err := runRepoCmd([]string{"delete", "alice/myrepo", "--yes"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if !strings.Contains(out, "Deleted alice/myrepo") && !strings.Contains(out, "已删除 alice/myrepo") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestRepoDeleteCmd_requiresYesInNonInteractiveMode(t *testing.T) {
	called := false
	_, err := runRepoCmd([]string{"delete", "alice/myrepo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected a --yes requirement, got %v", err)
	}
	if called {
		t.Fatal("delete request should not be sent without --yes")
	}
}

func TestRepoListCmd_empty(t *testing.T) {
	out, err := runRepoCmd([]string{"list"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.Repository{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No repositories found") && !strings.Contains(out, "未找到任何仓库") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestRepoCreateCmd(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			called = true
			json.NewEncoder(w).Encode(gitee.Repository{
				FullName: "alice/newrepo",
				HTMLURL:  "https://gitee.com/alice/newrepo",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runRepoCmd([]string{"create", "--name", "newrepo"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected POST request to create repo")
	}
	if !strings.Contains(out, "Created") && !strings.Contains(out, "已创建") {
		t.Errorf("expected creation message, got: %s", out)
	}
}

func TestRepoCreateCmd_missingName(t *testing.T) {
	_, err := runRepoCmd([]string{"create"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error when --name is missing")
	}
}

func TestRepoForkCmd(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/forks") {
			called = true
			json.NewEncoder(w).Encode(gitee.Repository{
				FullName: "bob/upstream",
				HTMLURL:  "https://gitee.com/bob/upstream",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runRepoCmd([]string{"fork", "alice/upstream"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected POST /forks request")
	}
	if !strings.Contains(out, "Forked to") && !strings.Contains(out, "已 Fork") {
		t.Errorf("expected fork message, got: %s", out)
	}
}

func TestSplitOwnerRepo(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"owner/repo", []string{"owner", "repo"}},
		{"owner/repo/extra", []string{"owner", "repo/extra"}},
		{"noslash", nil},
		{"/noleft", nil},
		{"noright/", nil},
	}
	for _, c := range cases {
		got := splitOwnerRepo(c.input)
		if c.want == nil && got != nil {
			t.Errorf("splitOwnerRepo(%q) = %v, want nil", c.input, got)
		} else if c.want != nil && (got == nil || got[0] != c.want[0] || got[1] != c.want[1]) {
			t.Errorf("splitOwnerRepo(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
