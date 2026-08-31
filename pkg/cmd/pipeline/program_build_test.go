package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// The program `build` commands are project-scoped: unlike the repo build commands
// they do NOT perform the billing open-check, so the request count is exactly the
// op chain (view/status/rebuild/last = 1, cancel = 1 with --yes, 0 without).

func programBuild(id, number int64, status string) giteego.PipelineBuildVO {
	return giteego.PipelineBuildVO{
		ID:          id,
		BuildNumber: number,
		Status:      status,
		FileName:    "prog-ci.yaml",
		Ref:         "master",
	}
}

// TestProgramBuildView asserts `program build view` renders the layout and hits
// the multi-source builds/{id} endpoint with pathBase 2/423.
func TestProgramBuildView(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, programBuild(9001, 42, "SUCCESS")))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "program", "build", "view", "9001", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if gotPathBase != "2/423" {
		t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
	}
	if !strings.Contains(out, "Build #42  [SUCCESS]") {
		t.Errorf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "Ref:    master") {
		t.Errorf("expected ref line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001") {
		t.Errorf("expected program build path, got %s", gotPaths[0])
	}
}

// TestProgramBuildViewNil asserts a nil build surfaces the not-found error.
func TestProgramBuildViewNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "view", "9001", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "build 9001 not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestProgramBuildStatus asserts the 状态树 rendering for a program build status.
func TestProgramBuildStatus(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PipelineBuildStatusVO{
			ID:        9001,
			Status:    "FAILED",
			StartTime: mustTime(t, "2026-08-17 10:00:00"),
			EndTime:   mustTime(t, "2026-08-17 10:02:45"),
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

	out, _, err := runPipelineCmd(t, f, "program", "build", "status", "9001", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #9001  [FAILED]") {
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
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001/status") {
		t.Errorf("expected program status path, got %s", gotPaths[0])
	}
}

// TestProgramBuildStatusNil asserts a nil status surfaces its error.
func TestProgramBuildStatusNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "status", "9001", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "build 9001 status not available") {
		t.Errorf("expected status error, got %v", err)
	}
}

// TestProgramBuildCancelYes asserts `program build cancel --yes` posts to
// /builds/{id}/cancel and prints the confirmation line.
func TestProgramBuildCancelYes(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "build", "cancel", "9001", "-E", "2", "-P", "423", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build 9001 cancelled\n") {
		t.Errorf("expected cancel line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001/cancel") {
		t.Errorf("expected POST cancel, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramBuildCancelNonInteractiveRequiresYes asserts the confirmation gate
// in a non-interactive context refuses to run without --yes and makes no request.
func TestProgramBuildCancelNonInteractiveRequiresYes(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "cancel", "9001", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected --yes requirement, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before confirmation, got %d", reqCount)
	}
}

// TestProgramBuildRebuild asserts `program build rebuild` posts and renders the
// rebuilt build.
func TestProgramBuildRebuild(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, programBuild(9002, 43, "WAITTING")))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "build", "rebuild", "9001", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #43  [WAITTING]") {
		t.Errorf("expected rebuilt build, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001/rebuild") {
		t.Errorf("expected rebuild path, got %s", gotPaths[0])
	}
}

// TestProgramBuildLast asserts `program build last` renders a build and passes
// the pipeline identifier query param.
func TestProgramBuildLast(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotQuery string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotQuery = r.URL.RawQuery
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, programBuild(9001, 42, "SUCCESS")))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "build", "last", "--pipeline", "706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #42  [SUCCESS]") {
		t.Errorf("expected build render, got:\n%s", out)
	}
	if !strings.Contains(gotQuery, "identifier=706") {
		t.Errorf("expected pipeline identifier query, got %q", gotQuery)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/last") {
		t.Errorf("expected last path, got %s", gotPaths[0])
	}
}

// TestProgramBuildLastNil asserts the empty-build message.
func TestProgramBuildLastNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "last", "--pipeline", "706", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "no build found for pipeline 706") {
		t.Errorf("expected no-build error, got %v", err)
	}
}

// TestProgramBuildLastMissingFlags asserts cobra's required-flag validation
// (sorted alphabetically).
func TestProgramBuildLastMissingFlags(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil)
	_, _, err := runPipelineCmd(t, f, "program", "build", "last", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "pipeline" not set`) {
		t.Errorf("expected required flags error, got %v", err)
	}
}

// TestProgramBuildIDArg covers the programBuildIDArg pure parser branches.
func TestProgramBuildIDArg(t *testing.T) {
	if _, err := programBuildIDArg(nil); err == nil || !strings.Contains(err.Error(), "build id is required") {
		t.Errorf("expected 'build id is required', got %v", err)
	}
	if _, err := programBuildIDArg([]string{"abc"}); err == nil || !strings.Contains(err.Error(), `invalid build id "abc"`) {
		t.Errorf("expected invalid build id, got %v", err)
	}
	if id, err := programBuildIDArg([]string{"9001"}); err != nil || id != 9001 {
		t.Errorf("programBuildIDArg(9001) = %d, %v", id, err)
	}
}

// TestProgramBuildMissingArg asserts cobra's arity validation fires first.
func TestProgramBuildMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "program", "build", "view", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}

func TestProgramBuildListRejectsInvalidPaging(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "list", "--pipeline", "706", "-E", "2", "-P", "423", "--page-size", "0")
	if err == nil || !strings.Contains(err.Error(), "--page-size must be at least 1") {
		t.Errorf("expected page-size error, got %v", err)
	}
	_, _, err = runPipelineCmd(t, f, "program", "build", "list", "--pipeline", "706", "-E", "2", "-P", "423", "--page", "0")
	if err == nil || !strings.Contains(err.Error(), "--page must be at least 1") {
		t.Errorf("expected page error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before validation, got %d", reqCount)
	}
}
