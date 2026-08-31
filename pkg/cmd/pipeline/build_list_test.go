package pipeline

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

func TestPipelineBuildList(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	var gotQuery url.Values
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineBuildSimpleVO]{
			Current:  1,
			PageSize: 10,
			Total:    2,
			Data: []giteego.PipelineBuildSimpleVO{
				{
					ID: 1079, BuildNumber: 8, FileName: "ci.yml", Ref: "master", Status: "FAILED",
					StartTime: mustTime(t, "2026-06-24 02:05:00"),
					Sources: []giteego.BuildSourceVO{
						{Type: "GiteeCode", Source: &giteego.BuildSourceDetail{Event: "merge_request_hooks", PrIID: 42, Message: "fix: CI"}},
					},
				},
				{
					ID: 1060, BuildNumber: 7, FileName: "ci.yml", Ref: "master", Status: "SUCCESS",
					StartTime: mustTime(t, "2026-06-23 10:00:00"),
					Sources: []giteego.BuildSourceVO{
						{Type: "GiteeCode", Source: &giteego.BuildSourceDetail{Branch: "feat/x", Message: "feat: add pipeline"}},
					},
				},
			},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "build", "list", "-R", "owner/repo",
		"--file", "ci.yml", "--ref", "master", "--status", "FAILED,SUCCESS",
		"--order", "create_time", "--sort", "desc")
	if err != nil {
		t.Fatal(err)
	}
	dump := strings.TrimSpace(out)
	// Table content.
	if !strings.Contains(dump, "1079") || !strings.Contains(dump, "FAILED") ||
		!strings.Contains(dump, "SUCCESS") || !strings.Contains(dump, "ci.yml") {
		t.Errorf("unexpected output:\n%s", out)
	}
	// SOURCE / COMMIT columns per the frontend rule: PR for merge_request_hooks,
	// branch for plain push; commit message from source.message.
	if !strings.Contains(dump, "PR #42") || !strings.Contains(dump, "fix: CI") {
		t.Errorf("expected PR source + commit, got:\n%s", out)
	}
	if !strings.Contains(dump, "feat/x") || !strings.Contains(dump, "feat: add pipeline") {
		t.Errorf("expected branch source + commit, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotPath, "/owner/repo/gitee-go/ipipe/rest/v5/pipelines/builds/history") {
		t.Errorf("expected history path, got %s", gotPath)
	}
	// Query params pass-through (order=field, sort=direction, statuses joined).
	if gotQuery.Get("order") != "create_time" || gotQuery.Get("sort") != "desc" {
		t.Errorf("expected order=create_time sort=desc, got %q %q", gotQuery.Get("order"), gotQuery.Get("sort"))
	}
	if gotQuery.Get("statuses") != "FAILED,SUCCESS" {
		t.Errorf("expected statuses=FAILED,SUCCESS, got %q", gotQuery.Get("statuses"))
	}
	if gotQuery.Get("fileName") != "ci.yml" || gotQuery.Get("ref") != "master" {
		t.Errorf("expected file/ref query, got %q %q", gotQuery.Get("fileName"), gotQuery.Get("ref"))
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
}

func TestPipelineBuildListEmpty(t *testing.T) {
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineBuildSimpleVO]{
			Current: 1, PageSize: 10, Total: 0, Data: []giteego.PipelineBuildSimpleVO{},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "build", "list", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	// English default locale in tests: "No build history found".
	if !strings.Contains(out, "No build history found") && !strings.Contains(out, "没有找到构建历史") {
		t.Errorf("expected empty-state message, got:\n%s", out)
	}
}

func TestPipelineBuildListRejectsInvalidPaging(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineBuildSimpleVO]{}))
	}, nil)

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"build", "list", "-R", "owner/repo", "--page-size", "0"}, "--page-size must be at least 1"},
		{[]string{"build", "list", "-R", "owner/repo", "--page", "-1"}, "--page must be at least 1"},
	} {
		_, _, err := runPipelineCmd(t, f, tc.args...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("args %v: expected %q, got %v", tc.args, tc.want, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before validation, got %d", reqCount)
	}
}

// A hostile/quirky server echo of pageSize=0 with total>0 must not panic the
// page footer math.
func TestPipelineBuildListZeroEchoedPageSize(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineBuildSimpleVO]{
			Current: 1, PageSize: 0, Total: 2,
			Data: []giteego.PipelineBuildSimpleVO{{ID: 1, BuildNumber: 1, FileName: "ci.yml", Ref: "master", Status: "SUCCESS"}},
		}))
	}, nil)
	if _, _, err := runPipelineCmd(t, f, "build", "list", "-R", "owner/repo"); err != nil {
		t.Fatal(err)
	}
}
