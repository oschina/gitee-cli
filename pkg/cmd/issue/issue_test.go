package issue

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func TestIssueListCmd_json(t *testing.T) {
	issues := []gitee.Issue{{Number: "42", Title: "JSON Issue"}}
	out, err := runIssueCmd([]string{"list", "-R", "owner/repo", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(issues)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("expected issue number in JSON output, got: %s", out)
	}
}

func TestIssueViewCmd_plainText(t *testing.T) {
	iss := gitee.Issue{Number: "5", Title: "View Me", State: "open", HTMLURL: "https://gitee.com/o/r/issues/5", Body: "Details here"}
	out, err := runIssueCmd([]string{"view", "5", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(iss)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "View Me") {
		t.Errorf("expected title in output, got: %s", out)
	}
	if !strings.Contains(out, "Details here") {
		t.Errorf("expected body in output, got: %s", out)
	}
}

func TestIssueViewCmd_json(t *testing.T) {
	iss := gitee.Issue{Number: "5", Title: "JSON Issue"}
	out, err := runIssueCmd([]string{"view", "5", "-R", "owner/repo", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(iss)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "JSON Issue") {
		t.Errorf("expected title in JSON output, got: %s", out)
	}
}

func TestIssueCommentCmd(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments") {
			called = true
			json.NewEncoder(w).Encode(gitee.Comment{ID: 77})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runIssueCmd([]string{"comment", "5", "-R", "owner/repo", "--body", "Good point"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected POST /comments request")
	}
	if !strings.Contains(out, "77") {
		t.Errorf("expected comment id in output, got: %s", out)
	}
}

func TestIssueCommentCmd_missingBody(t *testing.T) {
	_, err := runIssueCmd([]string{"comment", "5", "-R", "owner/repo"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error when --body is missing")
	}
}

func TestIssueCreateCmd_nonInteractive(t *testing.T) {
	created := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			created = true
			json.NewEncoder(w).Encode(gitee.Issue{Number: "99", HTMLURL: "https://gitee.com/o/r/issues/99"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewIssueCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs([]string{"create", "-R", "owner/repo", "--title", "New Issue"})
	err := root.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("expected POST create request")
	}
	if !strings.Contains(tf.Output(), "99") {
		t.Errorf("expected issue number in output, got: %s", tf.Output())
	}
}

func TestIssueCreateCmd_missingTitleNonInteractive(t *testing.T) {
	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()
	root := NewIssueCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs([]string{"create", "-R", "owner/repo"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error when title is missing in non-interactive mode")
	}
}

func TestIssueEditCmd_nonInteractive(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			called = true
			json.NewEncoder(w).Encode(gitee.Issue{Number: "7", Title: "Updated"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runIssueCmd([]string{"edit", "7", "-R", "owner/repo", "--title", "Updated"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected PATCH request to edit issue")
	}
	if !strings.Contains(out, "Updated issue #7") && !strings.Contains(out, "已更新 Issue #7") {
		t.Errorf("expected update message, got: %s", out)
	}
}

func TestIssueEditCmd_supportsClearingAndAdditionalFields(t *testing.T) {
	called := false
	_, err := runIssueCmd([]string{
		"edit", "ICX4FO", "-R", "owner/repo",
		"--body", "",
		"--assignee", "",
		"--labels", "",
		"--milestone", "12",
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		called = true
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"body", "assignee", "labels"} {
			if value, ok := payload[field]; !ok || value != "" {
				t.Errorf("expected explicit empty %s, got %#v", field, payload)
			}
		}
		if payload["milestone"] != float64(12) {
			t.Errorf("expected milestone=12, got %#v", payload["milestone"])
		}
		if _, ok := payload["title"]; ok {
			t.Errorf("title should be omitted, got %#v", payload)
		}
		json.NewEncoder(w).Encode(gitee.Issue{Number: "ICX4FO", HTMLURL: "https://gitee.com/owner/repo/issues/ICX4FO"})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected PATCH request")
	}
}

func TestIssueEditCmd_rejectsInvalidMilestone(t *testing.T) {
	_, err := runIssueCmd([]string{"edit", "ICX4FO", "-R", "owner/repo", "--milestone", "0"}, http.NotFoundHandler())
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected milestone validation error, got %v", err)
	}
}

func TestAssigneeLogin_noAssignee(t *testing.T) {
	iss := gitee.Issue{}
	got := assigneeLogin(iss)
	if got != "-" {
		t.Errorf("expected '-' for no assignee, got: %s", got)
	}
}

func TestAssigneeLogin_withAssignee(t *testing.T) {
	iss := gitee.Issue{Assignee: &gitee.User{Login: "dave"}}
	got := assigneeLogin(iss)
	if got != "dave" {
		t.Errorf("expected 'dave', got: %s", got)
	}
}

func TestLabelNames_noLabels(t *testing.T) {
	iss := gitee.Issue{}
	got := labelNames(iss)
	if got != "-" {
		t.Errorf("expected '-' for no labels, got: %s", got)
	}
}

func TestLabelNames_withLabels(t *testing.T) {
	iss := gitee.Issue{Labels: []gitee.Label{{Name: "bug"}, {Name: "feature"}}}
	got := labelNames(iss)
	if !strings.Contains(got, "bug") || !strings.Contains(got, "feature") {
		t.Errorf("expected label names, got: %s", got)
	}
}
