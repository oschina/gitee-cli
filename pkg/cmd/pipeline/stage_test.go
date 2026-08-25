package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// TestPipelineStageCancelBlockedWhenFinished verifies the running-state gate:
// cancelling a stage whose owning build is already finished is refused.
func TestPipelineStageCancelBlockedWhenFinished(t *testing.T) {
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, "/cancel") {
			// GetStageBuild for id 2278
			_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "FAILED"}))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/pipelines/builds/1079") {
			// GetBuild 1079 -> already finished
			_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1079, BuildNumber: 8, Status: "FAILED"}))
			return
		}
		// The cancel POST should never be reached; return success so a reach
		// would not go unnoticed as a hard failure.
		_, _ = w.Write(jsonBody(t, "ok"))
	}, &gotPathBase)

	out, errOut, err := runPipelineCmd(t, f, "build", "stage", "cancel", "2278", "-R", "owner/repo", "-y")
	if err == nil {
		t.Fatalf("expected an error for finished build, got none; stdout=%s stderr=%s", out, errOut)
	}
	if !strings.Contains(err.Error(), "already finished") {
		t.Errorf("expected 'already finished' error, got: %v", err)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
}

// TestPipelineStageCancelNonInteractiveRequiresYes verifies that in a
// non-interactive context the confirmation requires --yes.
func TestPipelineStageCancelNonInteractiveRequiresYes(t *testing.T) {
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, "/cancel") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "RUNNING"}))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/pipelines/builds/1079") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1079, BuildNumber: 8, Status: "RUNNING"}))
			return
		}
		_, _ = w.Write(jsonBody(t, "ok"))
	}, &gotPathBase)

	_, _, err := runPipelineCmd(t, f, "build", "stage", "cancel", "2278", "-R", "owner/repo")
	if err == nil {
		t.Fatal("expected --yes requirement error in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected --yes requirement, got: %v", err)
	}
}

// TestPipelineStageView asserts the stage view renders and performs the billing
// check first (repo = billing + stage = 2 requests).
func TestPipelineStageView(t *testing.T) {
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
		_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{
			ID:              2278,
			Name:            "build",
			Status:          "RUNNING",
			PipelineBuildID: 1079,
			StartTime:       mustTime(t, "2026-06-24 02:05:00"),
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "build", "stage", "view", "2278", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
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
	if !strings.Contains(out, "Start:   2026-06-24 02:05:00") {
		t.Errorf("expected start line, got:\n%s", out)
	}
	if !strings.Contains(out, "Duration -") {
		t.Errorf("expected duration line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+view), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines/stages/builds/2278") {
		t.Errorf("expected stage view path, got %s", gotPaths[1])
	}
}

// TestPipelineStageRetryContinue asserts the retry/continue side-effect ops pass
// the running-state gate then post to their endpoints (repo = billing + stage-get
// + build-get + op = 4 requests).
func TestPipelineStageRetryContinue(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantSfx string
		wantMsg string
	}{
		{name: "retry", args: []string{"build", "stage", "retry", "2278", "-R", "owner/repo", "-y"}, wantSfx: "/retry", wantMsg: "Stage 2278 retried\n"},
		{name: "continue", args: []string{"build", "stage", "continue", "2278", "-R", "owner/repo", "-y"}, wantSfx: "/continue", wantMsg: "Stage 2278 continued\n"},
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
				if strings.Contains(r.URL.Path, "billing") {
					_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
					return
				}
				if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, tt.wantSfx) {
					_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "RUNNING"}))
					return
				}
				if strings.HasSuffix(r.URL.Path, "/pipelines/builds/1079") {
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
			if len(gotPaths) != 4 {
				t.Fatalf("expected 4 requests (billing+stage+build+op), got %v", gotPaths)
			}
			if !strings.Contains(gotPaths[3], "/stages/builds/2278"+tt.wantSfx) {
				t.Errorf("expected op path %s, got %s", tt.wantSfx, gotPaths[3])
			}
		})
	}
}

// TestPipelineStageCancelHappy asserts the cancel happy path with the running
// gate and --yes (mirrors TestPipelineStageRetryContinue).
func TestPipelineStageCancelHappy(t *testing.T) {
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
		if strings.Contains(r.URL.Path, "/stages/builds/2278") && !strings.Contains(r.URL.Path, "/cancel") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineStageBuildVO{ID: 2278, PipelineBuildID: 1079, Status: "RUNNING"}))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/pipelines/builds/1079") {
			_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 1079, BuildNumber: 8, Status: "RUNNING"}))
			return
		}
		_, _ = w.Write(jsonBody(t, "ok"))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "build", "stage", "cancel", "2278", "-R", "owner/repo", "-y")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Stage 2278 cancelled\n") {
		t.Errorf("expected cancel line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 4 {
		t.Fatalf("expected 4 requests (billing+stage+build+cancel), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[3], "/stages/builds/2278/cancel") {
		t.Errorf("expected cancel path, got %s", gotPaths[3])
	}
}

// TestStageIDArg covers the stageIDArg pure parser branches.
func TestStageIDArg(t *testing.T) {
	if _, err := stageIDArg(nil); err == nil || !strings.Contains(err.Error(), "stage id is required") {
		t.Errorf("expected 'stage id is required', got %v", err)
	}
	if _, err := stageIDArg([]string{"abc"}); err == nil || !strings.Contains(err.Error(), `invalid stage id "abc"`) {
		t.Errorf("expected invalid stage id, got %v", err)
	}
	if id, err := stageIDArg([]string{"2278"}); err != nil || id != 2278 {
		t.Errorf("stageIDArg(2278) = %d, %v", id, err)
	}
}

// TestPipelineStageMissingArg asserts cobra's arity validation fires first.
func TestPipelineStageMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "build", "stage", "view", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}
