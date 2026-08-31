package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// The program `build job` side-effect ops (cancel/skip/retry/mark-success) run
// without a billing check and confirm via --yes in non-interactive contexts:
// 1 request each with --yes, 0 without.

// TestProgramJobOps verifies each side-effect op posts to its multi-source
// endpoint and prints its confirmation line.
func TestProgramJobOps(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		args    []string
		wantSfx string
		wantMsg string
	}{
		{name: "cancel", cmd: "cancel", wantSfx: "/stages/jobs/builds/2730/cancel", wantMsg: "Job 2730 cancelled\n"},
		{name: "skip", cmd: "skip", wantSfx: "/stages/jobs/builds/2730/skip", wantMsg: "Job 2730 skipped\n"},
		{name: "retry", cmd: "retry", wantSfx: "/stages/jobs/builds/2730/retry", wantMsg: "Job 2730 retried\n"},
		{name: "mark-success", cmd: "mark-success", wantSfx: "/stages/jobs/builds/2730/mark-as-success", wantMsg: "Job 2730 marked as success\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var gotPaths []string
			var gotPathBase string
			f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotPaths = append(gotPaths, r.URL.Path)
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.Write(jsonBody(t, "ok"))
			}, &gotPathBase)

			args := append([]string{"program", "build", "job", tt.cmd, "2730", "-E", "2", "-P", "423", "--yes"}, tt.args...)
			out, _, err := runPipelineCmd(t, f, args...)
			if err != nil {
				t.Fatalf("unexpected error: %v; stdout=%s", err, out)
			}
			if gotPathBase != "2/423" {
				t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
			}
			if !strings.Contains(out, tt.wantMsg) {
				t.Errorf("stdout = %q, want contains %q", out, tt.wantMsg)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(gotPaths) != 1 {
				t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
			}
			if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines"+tt.wantSfx) {
				t.Errorf("expected op path %s, got %s", tt.wantSfx, gotPaths[0])
			}
		})
	}
}

// TestProgramJobMarkSuccessBody asserts the mark-success POST carries the reason.
func TestProgramJobMarkSuccessBody(t *testing.T) {
	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "build", "job", "mark-success", "2730", "-E", "2", "-P", "423", "--yes", "--reason", "verified")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Job 2730 marked as success\n") {
		t.Errorf("expected message, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotBody, `"reason":"verified"`) {
		t.Errorf("expected reason in body, got %s", gotBody)
	}
}

// TestProgramJobNonInteractiveRequiresYes verifies that in a non-interactive
// context the confirmation requires --yes and no request is fired.
func TestProgramJobNonInteractiveRequiresYes(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{name: "cancel", cmd: "cancel"},
		{name: "skip", cmd: "skip"},
		{name: "retry", cmd: "retry"},
		{name: "mark-success", cmd: "mark-success"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var reqCount int
			f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				reqCount++
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.Write(jsonBody(t, "ok"))
			}, nil)

			_, _, err := runPipelineCmd(t, f, "program", "build", "job", tt.cmd, "2730", "-E", "2", "-P", "423")
			if err == nil || !strings.Contains(err.Error(), "--yes") {
				t.Errorf("expected --yes requirement, got %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if reqCount != 0 {
				t.Fatalf("expected no request before confirmation, got %d", reqCount)
			}
		})
	}
}

// TestProgramJobClientErrorSurfaced asserts a non-zero code from the op endpoint
// is surfaced as an error.
func TestProgramJobClientErrorSurfaced(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, 0))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "build", "job", "retry", "2730", "-E", "2", "-P", "423", "--yes")
	if err == nil {
		t.Fatal("expected an error for non-zero code")
	}
}

// TestProgramJobMissingArg asserts cobra's arity validation fires first.
func TestProgramJobMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "program", "build", "job", "cancel", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}

// TestProgramJobIDArg covers the shared jobIDArg pure parser branches.
func TestProgramJobIDArg(t *testing.T) {
	if _, err := jobIDArg(nil); err == nil || !strings.Contains(err.Error(), "job id is required") {
		t.Errorf("expected 'job id is required', got %v", err)
	}
	if _, err := jobIDArg([]string{"abc"}); err == nil || !strings.Contains(err.Error(), `invalid job id "abc"`) {
		t.Errorf("expected invalid job id, got %v", err)
	}
	if id, err := jobIDArg([]string{"2730"}); err != nil || id != 2730 {
		t.Errorf("jobIDArg(2730) = %d, %v", id, err)
	}
}
