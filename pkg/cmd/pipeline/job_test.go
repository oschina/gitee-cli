package pipeline

import (
	"net/http"
	"strings"
	"testing"
)

// TestPipelineJobOps covers the repo-scoped job side-effect commands. Each
// success run performs a billing status check followed by the op POST; the
// --yes flag skips the confirmation prompt.
func TestPipelineJobOps(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantSfx string
		wantMsg string
	}{
		{name: "cancel", args: []string{"build", "job", "cancel", "2730", "-R", "owner/repo", "-y"}, wantSfx: "/cancel", wantMsg: "Job 2730 cancelled\n"},
		{name: "skip", args: []string{"build", "job", "skip", "2730", "-R", "owner/repo", "-y"}, wantSfx: "/skip", wantMsg: "Job 2730 skipped\n"},
		{name: "retry", args: []string{"build", "job", "retry", "2730", "-R", "owner/repo", "-y"}, wantSfx: "/retry", wantMsg: "Job 2730 retried\n"},
		{name: "mark-success", args: []string{"build", "job", "mark-success", "2730", "-R", "owner/repo", "-y", "--reason", "verified"}, wantSfx: "/mark-as-success", wantMsg: "Job 2730 marked as success\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPathBase string
			var requestCount int
			f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(r.URL.Path, "billing") {
					_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
					return
				}
				_, _ = w.Write(jsonBody(t, "ok"))
			}, &gotPathBase)

			out, errOut, err := runPipelineCmd(t, f, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v; stdout=%s stderr=%s", err, out, errOut)
			}
			if requestCount != 2 {
				t.Errorf("expected 2 requests (billing + op), got %d", requestCount)
			}
			if gotPathBase != "owner/repo" {
				t.Errorf("pathBase = %q, want owner/repo", gotPathBase)
			}
			if !strings.Contains(out, tt.wantMsg) {
				t.Errorf("stdout = %q, want contains %q", out, tt.wantMsg)
			}
		})
	}
}

// TestPipelineJobMarkSuccessBody asserts mark-success sends the reason body.
func TestPipelineJobMarkSuccessBody(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/mark-as-success") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		body := readBody(t, r)
		want := `{"reason":"verified"}`
		if body != want {
			t.Errorf("body = %s, want %s", body, want)
		}
		_, _ = w.Write(jsonBody(t, "ok"))
	}, nil)

	if _, _, err := runPipelineCmd(t, f, "build", "job", "mark-success", "2730", "-R", "owner/repo", "-y", "--reason", "verified"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestPipelineJobNonInteractiveRequiresYes verifies --yes is mandatory for
// job side-effect ops in a non-interactive context.
func TestPipelineJobNonInteractiveRequiresYes(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "build", "job", "cancel", "2730", "-R", "owner/repo")
	if err == nil {
		t.Fatal("expected --yes requirement error in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected --yes requirement, got: %v", err)
	}
}

// TestJobIDArg covers the jobIDArg pure-parser error branches.
func TestJobIDArg(t *testing.T) {
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

// TestPipelineJobMissingArg asserts cobra's arity validation fires first.
func TestPipelineJobMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "build", "job", "cancel", "-R", "owner/repo", "-y")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}

// TestPipelineJobClientErrorEnsureServiceOpenRedirectsYAML asserts the job ID
// matcher does not accidentally swallow API errors (paths do not collide).
func TestPipelineJobClientErrorSurfaced(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write([]byte(`{"code":500,"msg":"job backend down","data":null}`))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "build", "job", "cancel", "2730", "-R", "owner/repo", "-y")
	if err == nil || !strings.Contains(err.Error(), "job backend down") {
		t.Errorf("expected code!=0 error, got %v", err)
	}
}
