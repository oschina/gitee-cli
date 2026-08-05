package issue

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runIssueCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewIssueCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestIssueListCmd_plainText(t *testing.T) {
	issues := []gitee.Issue{
		{Number: "I1", Title: "Cannot login", State: "open", User: gitee.User{Login: "alice"}},
	}
	out, err := runIssueCmd([]string{"list", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(issues)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Cannot login") {
		t.Errorf("expected issue title, got: %s", out)
	}
}

func TestIssueListCmd_empty(t *testing.T) {
	out, err := runIssueCmd([]string{"list", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.Issue{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No issues found") && !strings.Contains(out, "未找到任何 Issue") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestIssueCloseCmd(t *testing.T) {
	called := false
	out, err := runIssueCmd([]string{"close", "IJEE5", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			called = true
			var p gitee.UpdateIssueParams
			json.NewDecoder(r.Body).Decode(&p)
			if p.State != "closed" {
				t.Errorf("expected state=closed, got %s", p.State)
			}
			json.NewEncoder(w).Encode(gitee.Issue{Number: "IJEE5", State: "closed"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("PATCH not called")
	}
	if !strings.Contains(out, "Closed issue #IJEE5") && !strings.Contains(out, "已关闭 Issue #IJEE5") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestIssueReopenCmd(t *testing.T) {
	out, err := runIssueCmd([]string{"reopen", "IJEE2", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p gitee.UpdateIssueParams
		json.NewDecoder(r.Body).Decode(&p)
		if p.State != "open" {
			t.Errorf("expected state=open, got %s", p.State)
		}
		json.NewEncoder(w).Encode(gitee.Issue{Number: "IJEE2", State: "open"})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Reopened issue #IJEE2") && !strings.Contains(out, "已重新开启 Issue #IJEE2") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestIssueAssignCmd(t *testing.T) {
	out, err := runIssueCmd([]string{"assign", "IJEE3", "charlie", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p gitee.UpdateIssueParams
		json.NewDecoder(r.Body).Decode(&p)
		if p.Assignee != "charlie" {
			t.Errorf("expected assignee=charlie, got %s", p.Assignee)
		}
		json.NewEncoder(w).Encode(gitee.Issue{Number: "IJEE3"})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "charlie") {
		t.Errorf("unexpected output: %s", out)
	}
}
