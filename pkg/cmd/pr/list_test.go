package pr

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runPRCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewPRCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestPRListCmd_plainText(t *testing.T) {
	prs := []gitee.PullRequest{
		{Number: 1, Title: "Fix login bug", State: "open", User: gitee.PRUser{Login: "alice"}, Head: gitee.BranchRef{Ref: "fix/login"}, Base: gitee.BranchRef{Ref: "main"}},
	}
	out, err := runPRCmd([]string{"list", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(prs)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Fix login bug") {
		t.Errorf("expected PR title, got: %s", out)
	}
}

func TestPRListCmd_json(t *testing.T) {
	prs := []gitee.PullRequest{{Number: 42, Title: "JSON PR"}}
	out, err := runPRCmd([]string{"list", "-R", "owner/repo", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(prs)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("expected 42 in output, got: %s", out)
	}
}

func TestPRListCmd_filters(t *testing.T) {
	out, err := runPRCmd([]string{
		"list",
		"-R", "owner/repo",
		"--state", "all",
		"--head", "alice:feature",
		"--base", "main",
		"--sort", "updated",
		"--since", "2026-05-01T00:00:00Z",
		"--direction", "asc",
		"--milestone-number", "7",
		"--labels", "bug,performance",
		"--page", "3",
		"--per-page", "50",
		"--author", "alice",
		"--assignee", "bob",
		"--tester", "carol",
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		want := map[string]string{
			"state":            "all",
			"head":             "alice:feature",
			"base":             "main",
			"sort":             "updated",
			"since":            "2026-05-01T00:00:00Z",
			"direction":        "asc",
			"milestone_number": "7",
			"labels":           "bug,performance",
			"page":             "3",
			"per_page":         "50",
			"author":           "alice",
			"assignee":         "bob",
			"tester":           "carol",
		}
		for key, value := range want {
			if got := q.Get(key); got != value {
				t.Errorf("expected %s=%s, got %s", key, value, got)
			}
		}
		json.NewEncoder(w).Encode([]gitee.PullRequest{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No pull requests found") && !strings.Contains(out, "未找到任何 Pull Request") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestPRListCmd_empty(t *testing.T) {
	out, err := runPRCmd([]string{"list", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.PullRequest{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No pull requests found") && !strings.Contains(out, "未找到任何 Pull Request") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestPRCloseCmd(t *testing.T) {
	called := false
	out, err := runPRCmd([]string{"close", "5", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			called = true
			json.NewEncoder(w).Encode(gitee.PullRequest{Number: 5, State: "closed"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected PATCH request")
	}
	if !strings.Contains(out, "Closed PR #5") && !strings.Contains(out, "已关闭 PR #5") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestPRReopenCmd(t *testing.T) {
	out, err := runPRCmd([]string{"reopen", "3", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var p gitee.UpdatePullParams
			json.NewDecoder(r.Body).Decode(&p)
			if p.State != "open" {
				t.Errorf("expected state=open, got %s", p.State)
			}
			json.NewEncoder(w).Encode(gitee.PullRequest{Number: 3, State: "open"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Reopened PR #3") && !strings.Contains(out, "已重新开启 PR #3") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestPRMergeCmd(t *testing.T) {
	called := false
	out, err := runPRCmd([]string{"merge", "7", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/merge") {
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
		t.Error("expected PUT /merge request")
	}
	if !strings.Contains(out, "Merged PR #7") && !strings.Contains(out, "已合并 PR #7") {
		t.Errorf("unexpected output: %s", out)
	}
}
