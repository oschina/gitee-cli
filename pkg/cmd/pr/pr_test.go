package pr

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	gitpkg "gitee.com/oschina/gitee-cli/pkg/git"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func TestPRViewCmd_plainText(t *testing.T) {
	pr := gitee.PullRequest{
		Number: 10,
		Title:  "My Feature",
		State:  "open",
		User:   gitee.PRUser{Login: "dev"},
		Head:   gitee.BranchRef{Ref: "feat/x"},
		Base:   gitee.BranchRef{Ref: "main"},
		Body:   "Some description",
	}
	out, err := runPRCmd([]string{"view", "10", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(pr)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "My Feature") {
		t.Errorf("expected title in output, got: %s", out)
	}
	if !strings.Contains(out, "dev") {
		t.Errorf("expected author in output, got: %s", out)
	}
	if !strings.Contains(out, "feat/x") {
		t.Errorf("expected head branch in output, got: %s", out)
	}
}

func TestPRViewCmd_json(t *testing.T) {
	pr := gitee.PullRequest{Number: 10, Title: "JSON PR"}
	out, err := runPRCmd([]string{"view", "10", "-R", "owner/repo", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(pr)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "JSON PR") {
		t.Errorf("expected title in JSON output, got: %s", out)
	}
}

func TestPRViewCmd_invalidNumber(t *testing.T) {
	_, err := runPRCmd([]string{"view", "notanumber", "-R", "owner/repo"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error for invalid PR number")
	}
}

func TestPRCommentCmd(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments") {
			called = true
			json.NewEncoder(w).Encode(gitee.Comment{ID: 55})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runPRCmd([]string{"comment", "10", "-R", "owner/repo", "--body", "LGTM"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected POST /comments request")
	}
	if !strings.Contains(out, "55") {
		t.Errorf("expected comment id in output, got: %s", out)
	}
}

func TestPRCommentCmd_missingBody(t *testing.T) {
	_, err := runPRCmd([]string{"comment", "10", "-R", "owner/repo"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error when --body is missing")
	}
}

func TestPRCommentCmd_invalidNumber(t *testing.T) {
	_, err := runPRCmd([]string{"comment", "abc", "-R", "owner/repo", "--body", "hi"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error for invalid PR number")
	}
}

func TestPRReviewCmd(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/review") {
			called = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runPRCmd([]string{"review", "11", "-R", "owner/repo"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected POST /review request")
	}
	if !strings.Contains(out, "Approved PR #11") && !strings.Contains(out, "已审批 PR #11") {
		t.Errorf("expected approval message, got: %s", out)
	}
}

func TestPRReviewCmd_force(t *testing.T) {
	var requestBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/review") {
			body, _ := io.ReadAll(r.Body)
			requestBody = string(body)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := runPRCmd([]string{"review", "11", "-R", "owner/repo", "--force"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(requestBody, `"force":true`) {
		t.Errorf("expected force=true in request body, got: %s", requestBody)
	}
}

func TestPRReviewCmd_withBody(t *testing.T) {
	reviewCalled := false
	commentCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/review") {
			reviewCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comments") {
			commentCalled = true
			json.NewEncoder(w).Encode(gitee.Comment{ID: 99})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runPRCmd([]string{"review", "11", "-R", "owner/repo", "--body", "LGTM"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !reviewCalled {
		t.Error("expected POST /review to be called")
	}
	if !commentCalled {
		t.Error("expected POST /comments to be called")
	}
	if !strings.Contains(out, "#11") {
		t.Errorf("expected PR number in output, got: %s", out)
	}
}

func TestPRReviewCmd_commentFailureReturnsError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/review") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/comments") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"comment unavailable"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := runPRCmd([]string{"review", "11", "-R", "owner/repo", "--body", "LGTM"}, handler)
	if err == nil || !strings.Contains(err.Error(), "was approved") {
		t.Fatalf("expected partial-success error, got %v", err)
	}
}

func TestPRReviewCmd_invalidNumber(t *testing.T) {
	_, err := runPRCmd([]string{"review", "abc", "-R", "owner/repo"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error for invalid PR number")
	}
}

func TestPRCreateCmd_nonInteractive(t *testing.T) {
	created := false
	var params gitee.CreatePullParams
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			created = true
			if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
				t.Errorf("decode create params: %v", err)
			}
			json.NewEncoder(w).Encode(gitee.PullRequest{Number: 99, HTMLURL: "https://gitee.com/owner/repo/pulls/99"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewPRCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs([]string{
		"create", "-R", "owner/repo",
		"--title", "New PR",
		"--head", "feat/new",
		"--base", "main",
		"--assignees", "alice,bob",
		"--testers", "carol",
	})
	err := root.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("expected POST create request")
	}
	if params.Assignees != "alice,bob" {
		t.Errorf("expected assignees=alice,bob, got %q", params.Assignees)
	}
	if params.Testers != "carol" {
		t.Errorf("expected testers=carol, got %q", params.Testers)
	}
	if !strings.Contains(tf.Output(), "99") {
		t.Errorf("expected PR number in output, got: %s", tf.Output())
	}
}

func TestPromptPRBaseBranch(t *testing.T) {
	got, err := promptPRBaseBranch("feature/new", "main", func(prompt, defaultValue string, useTUI bool) (string, error) {
		if !strings.Contains(prompt, "feature/new") {
			t.Errorf("expected prompt to identify the head branch, got %q", prompt)
		}
		if defaultValue != "main" {
			t.Errorf("expected inferred base as default, got %q", defaultValue)
		}
		if useTUI {
			t.Error("expected plain prompt mode")
		}
		return "release", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "release" {
		t.Fatalf("expected selected base branch, got %q", got)
	}
}

func TestPRCreateCmd_rejectsSingularAssigneeFlag(t *testing.T) {
	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()
	root := NewPRCmd(tf.Factory)
	root.SetArgs([]string{
		"create", "-R", "owner/repo",
		"--title", "New PR",
		"--head", "feat/new",
		"--assignee", "alice",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown flag: --assignee") {
		t.Fatalf("expected singular flag to be rejected, got %v", err)
	}
}

func TestPRCreateCmd_missingTitleNonInteractive(t *testing.T) {
	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()
	root := NewPRCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs([]string{"create", "-R", "owner/repo", "--head", "feat/x"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error when title is missing in non-interactive mode")
	}
}

func TestPRDiffCmd_plainText(t *testing.T) {
	status := "modified"
	files := []gitee.DiffFile{
		{
			Filename: "file.go",
			Status:   &status,
			Patch: gitee.PatchInfo{
				OldPath: "file.go",
				NewPath: "file.go",
				Diff:    "@@ -1 +1 @@\n-old line\n+new line\n",
			},
		},
	}
	out, err := runPRCmd([]string{"diff", "10", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "diff --git") {
		t.Errorf("expected diff content in output, got: %s", out)
	}
}

func TestPRDiffCmd_json(t *testing.T) {
	status := "modified"
	files := []gitee.DiffFile{
		{
			Filename:  "file.go",
			Status:    &status,
			Additions: "1",
			Deletions: "1",
			Patch: gitee.PatchInfo{
				OldPath: "file.go",
				NewPath: "file.go",
				Diff:    "@@ -1 +1 @@\n-old line\n+new line\n",
			},
		},
	}
	out, err := runPRCmd([]string{"diff", "10", "-R", "owner/repo", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}))
	if err != nil {
		t.Fatal(err)
	}
	var got []gitee.DiffFile
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected pure JSON output, got parse error %v; raw: %s", err, out)
	}
	if len(got) != 1 || got[0].Filename != "file.go" {
		t.Errorf("expected 1 file 'file.go', got: %s", out)
	}
	if strings.Contains(out, "diff --git") {
		t.Errorf("JSON mode should not emit plain-text diff, got: %s", out)
	}
}

func TestPRDiffCmd_jsonFields(t *testing.T) {
	files := []gitee.DiffFile{{Filename: "a.go", Additions: "3", Deletions: "0"}}
	out, err := runPRCmd([]string{"diff", "10", "-R", "owner/repo", "--json=filename,additions"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(files)
	}))
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected pure JSON output, got parse error %v; raw: %s", err, out)
	}
	if len(got) != 1 || got[0]["filename"] != "a.go" {
		t.Errorf("expected selected field filename=a.go, got: %s", out)
	}
	if _, hasDeletions := got[0]["deletions"]; hasDeletions {
		t.Errorf("field selection should exclude 'deletions', got: %s", out)
	}
}

func TestPRDiffCmd_invalidNumber(t *testing.T) {
	_, err := runPRCmd([]string{"diff", "notanumber", "-R", "owner/repo"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error for invalid PR number")
	}
}

func TestPickRemote_first(t *testing.T) {
	remotes := []gitpkg.Remote{{Name: "origin", URL: "https://gitee.com/o/r.git"}}
	got, err := pickRemote(remotes, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "origin" {
		t.Errorf("expected 'origin', got %q", got.Name)
	}
}

func TestPickRemote_named(t *testing.T) {
	remotes := []gitpkg.Remote{
		{Name: "origin", URL: "https://gitee.com/o/r1.git"},
		{Name: "upstream", URL: "https://gitee.com/o/r2.git"},
	}
	got, err := pickRemote(remotes, "upstream")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "upstream" {
		t.Errorf("expected 'upstream', got %q", got.Name)
	}
}

func TestPickRemote_notFound(t *testing.T) {
	remotes := []gitpkg.Remote{{Name: "origin", URL: "https://gitee.com/o/r.git"}}
	_, err := pickRemote(remotes, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent remote")
	}
}

func TestPickRemote_empty(t *testing.T) {
	_, err := pickRemote([]gitpkg.Remote{}, "")
	if err == nil {
		t.Error("expected error when no remotes available")
	}
}
