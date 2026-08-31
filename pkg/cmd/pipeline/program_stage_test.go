package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// The program `build stage` commands are project-scoped and do NOT perform the
// billing open-check. The side-effect ops gate on the owning build's running
// state exactly like the repo stage ops: cancel/retry/continue = stage-get +
// build-get + op (3 requests), view = 1 request.

func TestProgramStageView(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PipelineStageBuildVO{
			ID:              2278,
			Name:            "build",
			Status:          "RUNNING",
			PipelineBuildID: 1079,
			StartTime:       mustTime(t, "2026-08-17 10:00:00"),
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "program", "build", "stage", "view", "2278", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if gotPathBase != "2/423" {
		t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
	}
	if !strings.Contains(out, "Stage    2278  [RUNNING]") {
		t.Errorf("expected stage header, got:\n%s", out)
	}
	if !strings.Contains(out, "Name:    build") {
		t.Errorf("expected name line, got:\n%s", out)
	}
	if !strings.Contains(out, "Build:   1079") {
		t.Errorf("expected build line, got:\n%s", out)
	}
	if !strings.Contains(out, "Duration -") {
		t.Errorf("expected duration line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/builds/2278") {
		t.Errorf("expected program stage path, got %s", gotPaths[0])
	}
}

// TestProgramStageViewNil asserts a nil stage surfaces the not-found error.
func TestProgramStageViewNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "stage", "view", "2278", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "stage 2278 not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestProgramStageRetryContinue asserts the retry/continue side-effect ops pass
// the running-state gate then post to their endpoints (no billing:
// stage-get + build-get + op = 3 requests).
func TestProgramStageRetryContinue(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantSfx string
		wantMsg string
	}{
		{name: "retry", args: []string{"program", "build", "stage", "retry", "2278", "-E", "2", "-P", "423", "--yes"}, wantSfx: "/retry", wantMsg: "Stage 2278 retried\n"},
		{name: "continue", args: []string{"program", "build", "stage", "continue", "2278", "-E", "2", "-P", "423", "--yes"}, wantSfx: "/continue", wantMsg: "Stage 2278 continued\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var gotPaths []string
			f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotPaths = append(gotPaths, r.URL.Path)
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, tt.wantSfx) {
					_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "RUNNING"}))
					return
				}
				if strings.Contains(r.URL.Path, "/pipelines/builds/1079") {
					_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1079, BuildNumber: 8, Status: "RUNNING"}))
					return
				}
				_, _ = w.Write(jsonBody(t, "ok"))
			}, nil)

			out, _, err := runPipelineCmd(t, f, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v; stdout=%s", err, out)
			}
			if !strings.Contains(out, tt.wantMsg) {
				t.Errorf("stdout = %q, want contains %q", out, tt.wantMsg)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(gotPaths) != 3 {
				t.Fatalf("expected 3 requests (stage+build+op), got %v", gotPaths)
			}
			if !strings.Contains(gotPaths[0], "/stages/builds/2278") || !strings.Contains(gotPaths[0], "/multi-source/") {
				t.Errorf("expected stage-get path, got %s", gotPaths[0])
			}
			if !strings.Contains(gotPaths[1], "/pipelines/builds/1079") {
				t.Errorf("expected build-get path, got %s", gotPaths[1])
			}
			if !strings.Contains(gotPaths[2], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/builds/2278"+tt.wantSfx) {
				t.Errorf("expected op path %s, got %s", tt.wantSfx, gotPaths[2])
			}
		})
	}
}

// TestProgramStageCancelHappy asserts the cancel happy path with the running gate.
func TestProgramStageCancelHappy(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, "/cancel") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "RUNNING"}))
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/builds/1079") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1079, BuildNumber: 8, Status: "RUNNING"}))
			return
		}
		_, _ = w.Write(jsonBody(t, "ok"))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "build", "stage", "cancel", "2278", "-E", "2", "-P", "423", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Stage 2278 cancelled\n") {
		t.Errorf("expected cancel line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 3 {
		t.Fatalf("expected 3 requests (stage+build+cancel), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[2], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/builds/2278/cancel") {
		t.Errorf("expected cancel path, got %s", gotPaths[2])
	}
}

// TestProgramStageCancelNonInteractiveRequiresYes verifies the --yes gate fires
// before the op but AFTER the running-state gate (2 requests, no op).
func TestProgramStageCancelNonInteractiveRequiresYes(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, "/cancel") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "RUNNING"}))
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/builds/1079") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1079, BuildNumber: 8, Status: "RUNNING"}))
			return
		}
		_, _ = w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "stage", "cancel", "2278", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected --yes requirement, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	// The running gate runs before the confirm, so stage+build are fetched but the
	// cancel POST must never fire.
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (stage+build, no op), got %v", gotPaths)
	}
}

// TestProgramStageCancelBlockedWhenFinished verifies the running-state gate:
// cancelling a stage whose owning build is already finished is refused.
func TestProgramStageCancelBlockedWhenFinished(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, "/cancel") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "FAILED"}))
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/builds/1079") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1079, BuildNumber: 8, Status: "FAILED"}))
			return
		}
		_, _ = w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "stage", "cancel", "2278", "-E", "2", "-P", "423", "--yes")
	if err == nil {
		t.Fatal("expected an error for finished build, got none")
	}
	if !strings.Contains(err.Error(), "already finished") {
		t.Errorf("expected 'already finished' error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 2 {
		t.Fatalf("expected 2 requests (stage+build, no op), got %d", reqCount)
	}
}

// TestProgramStageMissingArg asserts cobra's arity validation fires first.
func TestProgramStageMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "program", "build", "stage", "view", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}
