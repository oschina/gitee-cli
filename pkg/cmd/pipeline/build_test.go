package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// TestPipelineBuildViewRenders asserts `build view` renders the full layout and
// performs the billing check first (repo = billing + GetBuild = 2 requests).
func TestPipelineBuildViewRenders(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{
			ID:          1079,
			BuildNumber: 8,
			Status:      "SUCCESS",
			FileName:    "ci.yml",
			Ref:         "master",
			StartTime:   mustTime(t, "2026-06-24 02:05:00"),
			EndTime:     mustTime(t, "2026-06-24 02:05:45"),
			Stages: []giteego.PipelineStageBuildVO{
				{Name: "build", Status: "SUCCESS"},
			},
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "build", "view", "1079", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #8  [SUCCESS]") {
		t.Errorf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "Start:  2026-06-24 02:05:00") {
		t.Errorf("expected start line, got:\n%s", out)
	}
	if !strings.Contains(out, "45s") {
		t.Errorf("expected duration, got:\n%s", out)
	}
	if !strings.Contains(out, "build  [SUCCESS]") {
		t.Errorf("expected stage line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+view), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines/builds/1079") {
		t.Errorf("expected view path, got %s", gotPaths[1])
	}
}

// TestPipelineBuildViewNil asserts a nil build surfaces the not-found error.
func TestPipelineBuildViewNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "build", "view", "1079", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), "build 1079 not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestPipelineBuildStatusTree asserts the 状态树 rendering with stage/job rows.
func TestPipelineBuildStatusTree(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildStatusVO{
			ID:        1079,
			Status:    "FAILED",
			StartTime: mustTime(t, "2026-06-24 02:05:00"),
			EndTime:   mustTime(t, "2026-06-24 02:05:45"),
			Stages: []giteego.PipelineStageStatusVO{
				{
					Name:   "build",
					Status: "FAILED",
					Jobs: [][]giteego.PipelineJobStatusVO{
						{{Name: "compile", Status: "FAILED"}},
					},
				},
			},
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "build", "status", "1079", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #1079  [FAILED]") {
		t.Errorf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "Status tree") || !strings.Contains(out, "------") {
		t.Errorf("expected status tree header, got:\n%s", out)
	}
	if !strings.Contains(out, "└─ build  [FAILED]") {
		t.Errorf("expected stage line, got:\n%s", out)
	}
	if !strings.Contains(out, `    ├─compile  [FAILED]`) {
		t.Errorf("expected job line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+status), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines/builds/1079/status") {
		t.Errorf("expected status path, got %s", gotPaths[1])
	}
}

// TestPipelineBuildStatusNil asserts a nil status surfaces its error.
func TestPipelineBuildStatusNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "build", "status", "1079", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), "build 1079 status not available") {
		t.Errorf("expected status error, got %v", err)
	}
}

// TestPipelineBuildCancel asserts `build cancel` (no --yes flag exists here)
// posts to /cancel and prints the confirmation line.
func TestPipelineBuildCancel(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		gotMethod = r.Method
		_, _ = w.Write(jsonBody(t, "ok"))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "build", "cancel", "1079", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build 1079 cancelled\n") {
		t.Errorf("expected cancel line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+cancel), got %v", gotPaths)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotPaths[1], "/builds/1079/cancel") {
		t.Errorf("expected POST cancel, method=%s path=%s", gotMethod, gotPaths[1])
	}
}

// TestPipelineBuildRebuild asserts `build rebuild` posts and renders the build.
func TestPipelineBuildRebuild(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1090, BuildNumber: 9, Status: "WAITTING", FileName: "ci.yml", Ref: "master"}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "build", "rebuild", "1079", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #9  [WAITTING]") {
		t.Errorf("expected rebuilt build, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+rebuild), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/builds/1079/rebuild") {
		t.Errorf("expected rebuild path, got %s", gotPaths[1])
	}
}

// TestPipelineBuildLast asserts `build last` renders a build and passes the
// ref/file query params.
func TestPipelineBuildLast(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotQuery string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotQuery = r.URL.RawQuery
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1090, BuildNumber: 9, Status: "SUCCESS", FileName: "ci.yml", Ref: "master"}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "build", "last", "-R", "owner/repo", "--file", "ci.yml", "--ref", "master")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #9  [SUCCESS]") {
		t.Errorf("expected build render, got:\n%s", out)
	}
	if !strings.Contains(gotQuery, "fileName=ci.yml") || !strings.Contains(gotQuery, "ref=master") {
		t.Errorf("expected fileName+ref query, got %q", gotQuery)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+last), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/builds/last") {
		t.Errorf("expected last path, got %s", gotPaths[1])
	}
}

// TestPipelineBuildLastNil asserts the empty-build message.
func TestPipelineBuildLastNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "build", "last", "-R", "owner/repo", "--file", "ci.yml", "--ref", "master")
	if err == nil || !strings.Contains(err.Error(), "no build found for ci.yml @ master") {
		t.Errorf("expected no-build error, got %v", err)
	}
}

// TestPipelineBuildLastMissingFlags asserts cobra's required-flag validation
// (sorted alphabetically).
func TestPipelineBuildLastMissingFlags(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil)
	_, _, err := runPipelineCmd(t, f, "build", "last", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "file", "ref" not set`) {
		t.Errorf("expected required flags error, got %v", err)
	}
}

// TestBuildIDArg covers the buildIDArg pure parser branches.
func TestBuildIDArg(t *testing.T) {
	if _, err := buildIDArg(nil); err == nil || !strings.Contains(err.Error(), "build id is required") {
		t.Errorf("expected 'build id is required', got %v", err)
	}
	if _, err := buildIDArg([]string{"abc"}); err == nil || !strings.Contains(err.Error(), `invalid build id "abc"`) {
		t.Errorf("expected invalid build id, got %v", err)
	}
	if id, err := buildIDArg([]string{"2730"}); err != nil || id != 2730 {
		t.Errorf("buildIDArg(2730) = %d, %v", id, err)
	}
}

// TestPipelineBuildMissingArg asserts cobra's arity validation fires first.
func TestPipelineBuildMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "build", "view", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}

// TestBuildHelpers covers the pure formatting/source helpers in build.go.
func TestBuildHelpers(t *testing.T) {
	// isTerminalBuildStatus
	for _, s := range []string{"SUCCESS", "FAILED", "CANCELLED", "SKIPPED", "TIMEOUT"} {
		if !isTerminalBuildStatus(s) {
			t.Errorf("expected %s terminal", s)
		}
	}
	for _, s := range []string{"RUNNING", "WAITTING", ""} {
		if isTerminalBuildStatus(s) {
			t.Errorf("expected %q non-terminal", s)
		}
	}

	// timeStr: nil/zero -> "-", value -> formatted.
	if got := timeStr(nil); got != "-" {
		t.Errorf("timeStr(nil) = %q, want -", got)
	}
	if got := timeStr(&giteego.FlexTime{}); got != "-" {
		t.Errorf("timeStr(zero) = %q, want -", got)
	}
	start := mustTime(t, "2026-06-24 02:05:00")
	if got := timeStr(start); got != "2026-06-24 02:05:00" {
		t.Errorf("timeStr(value) = %q, want 2026-06-24 02:05:00", got)
	}

	// durationStr: nil inputs, negative span, sub-minute, multi-minute.
	if got := durationStr(nil, start); got != "-" {
		t.Errorf("durationStr(nil) = %q, want -", got)
	}
	endNeg := &giteego.FlexTime{Time: start.Time.Add(-time.Minute)}
	if got := durationStr(start, endNeg); got != "-" {
		t.Errorf("durationStr(negative) = %q, want -", got)
	}
	end45 := &giteego.FlexTime{Time: start.Time.Add(45 * time.Second)}
	if got := durationStr(start, end45); got != "45s" {
		t.Errorf("durationStr(45s) = %q, want 45s", got)
	}
	end74 := &giteego.FlexTime{Time: start.Time.Add(74*time.Minute + 13*time.Second)}
	if got := durationStr(start, end74); got != "74m13s" {
		t.Errorf("durationStr(74m13s) = %q, want 74m13s", got)
	}

	// formatParamValue: nil, string, nested slice, scalar.
	if got := formatParamValue(nil); got != "" {
		t.Errorf("formatParamValue(nil) = %q, want empty", got)
	}
	if got := formatParamValue("x"); got != "x" {
		t.Errorf("formatParamValue(str) = %q, want x", got)
	}
	if got := formatParamValue([]interface{}{"a", "b", 1}); got != "a, b, 1" {
		t.Errorf("formatParamValue(slice) = %q, want 'a, b, 1'", got)
	}
	if got := formatParamValue(42); got != "42" {
		t.Errorf("formatParamValue(int) = %q, want 42", got)
	}

	// oneLine: newline collapse + truncation.
	if got := oneLine("a\nb"); got != `a\nb` {
		t.Errorf("oneLine(newline) = %q, want a\\nb", got)
	}
	if got := oneLine(strings.Repeat("x", 121)); !strings.HasSuffix(got, "…") || strings.Count(got, "x") != 120 {
		t.Errorf("oneLine(truncate) = %q", got)
	}

	// isPRSource: PR hook events only.
	if !isPRSource(&giteego.BuildSourceDetail{Event: "merge_request_hooks"}) {
		t.Error("expected merge_request_hooks to be a PR source")
	}
	if !isPRSource(&giteego.BuildSourceDetail{Event: "note_hooks"}) {
		t.Error("expected note_hooks to be a PR source")
	}
	if isPRSource(&giteego.BuildSourceDetail{Event: "push_hooks"}) || isPRSource(&giteego.BuildSourceDetail{Branch: "master"}) || isPRSource(nil) {
		t.Error("expected non-PR events to be rejected")
	}

	// historySource: PR #n beats branch, branch beats "-".
	pr := []giteego.BuildSourceVO{{Source: &giteego.BuildSourceDetail{Event: "merge_request_hooks", PrIID: 12}}}
	if got := historySource(pr); got != "PR #12" {
		t.Errorf("historySource(PR) = %q, want PR #12", got)
	}
	branch := []giteego.BuildSourceVO{{Source: &giteego.BuildSourceDetail{Branch: "develop"}}}
	if got := historySource(branch); got != "develop" {
		t.Errorf("historySource(branch) = %q, want develop", got)
	}
	if got := historySource(nil); got != "-" {
		t.Errorf("historySource(nil) = %q, want -", got)
	}

	// historyCommit: first message, else "-".
	msgs := []giteego.BuildSourceVO{{Source: &giteego.BuildSourceDetail{Message: "ci: update"}}}
	if got := historyCommit(msgs); got != "ci: update" {
		t.Errorf("historyCommit = %q, want 'ci: update'", got)
	}
	if got := historyCommit(nil); got != "-" {
		t.Errorf("historyCommit(nil) = %q, want -", got)
	}
}
