package release

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runReleaseCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewReleaseCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestReleaseListCmd(t *testing.T) {
	releases := []gitee.Release{
		{ID: 1, TagName: "v1.0.0", Name: "First Release", CreatedAt: time.Now()},
		{ID: 2, TagName: "v2.0.0", Name: "Second Release", CreatedAt: time.Now()},
	}
	out, err := runReleaseCmd([]string{"list", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releases)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "v1.0.0") {
		t.Errorf("expected tag name, got: %s", out)
	}
	if !strings.Contains(out, "v2.0.0") {
		t.Errorf("expected second tag, got: %s", out)
	}
}

func TestReleaseListCmd_empty(t *testing.T) {
	out, err := runReleaseCmd([]string{"list", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.Release{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No releases found") && !strings.Contains(out, "未找到任何 Release") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestReleaseListCmd_emptyJSON(t *testing.T) {
	out, err := runReleaseCmd([]string{"list", "-R", "owner/repo", "--json"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.Release{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out); got != "[]" {
		t.Fatalf("expected an empty JSON array, got %q", got)
	}
}

func TestReleaseViewCmd_byID(t *testing.T) {
	r := gitee.Release{ID: 3, TagName: "v3.0.0", Name: "Third", Author: gitee.User{Login: "alice"}, CreatedAt: time.Now()}
	out, err := runReleaseCmd([]string{"view", "3", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r2 *http.Request) {
		if strings.Contains(r2.URL.Path, "/releases/3") {
			json.NewEncoder(w).Encode(r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "v3.0.0") {
		t.Errorf("expected tag name, got: %s", out)
	}
}

func TestReleaseViewCmd_byTag(t *testing.T) {
	r := gitee.Release{ID: 5, TagName: "v5.0.0", Name: "Fifth", Author: gitee.User{Login: "bob"}, CreatedAt: time.Now()}
	out, err := runReleaseCmd([]string{"view", "v5.0.0", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r2 *http.Request) {
		if strings.Contains(r2.URL.Path, "/tags/v5.0.0") {
			json.NewEncoder(w).Encode(r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "v5.0.0") {
		t.Errorf("expected tag name, got: %s", out)
	}
}

func TestReleaseDeleteCmd(t *testing.T) {
	called := false
	out, err := runReleaseCmd([]string{"delete", "4", "-R", "owner/repo", "--yes"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if !strings.Contains(out, "Deleted release 4") && !strings.Contains(out, "已删除 Release 4") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestReleaseDeleteCmd_requiresYesInNonInteractiveMode(t *testing.T) {
	called := false
	_, err := runReleaseCmd([]string{"delete", "v1.0.0", "-R", "owner/repo"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected a --yes requirement, got %v", err)
	}
	if called {
		t.Fatal("release lookup or delete should not run without --yes")
	}
}
