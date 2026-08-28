package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// TestPipelineRunParams asserts run triggers a build and prints params.
func TestPipelineRunParams(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		gotBody = readBody(t, r)
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{
			ID:          7,
			BuildNumber: 12,
			Status:      "WAITTING",
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "run", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml", "-p", "BRANCH=main", "-p", "TAG=v1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #12 triggered") {
		t.Errorf("expected triggered line, got:\n%s", out)
	}
	if !strings.Contains(out, "ID:     7") {
		t.Errorf("expected ID line, got:\n%s", out)
	}
	if !strings.Contains(out, "Status: WAITTING") {
		t.Errorf("expected status line, got:\n%s", out)
	}
	if !strings.Contains(out, "BRANCH=main") || !strings.Contains(out, "TAG=v1") {
		t.Errorf("expected params section, got:\n%s", out)
	}
	// Params must be sent in the trigger body.
	if !strings.Contains(gotBody, `"name":"BRANCH","value":"main"`) {
		t.Errorf("expected BRANCH in body, got %s", gotBody)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+trigger), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines/builds") {
		t.Errorf("expected trigger path, got %s", gotPaths[1])
	}
}

// TestPipelineRunNormalizeFileName asserts a path-style --file (e.g.
// .workflow/ci.yml) is reduced to its base name before being sent to the
// backend, and that the note line surfaces the normalized name.
func TestPipelineRunNormalizeFileName(t *testing.T) {
	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{
			ID:          7,
			BuildNumber: 12,
			Status:      "WAITTING",
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "run", "-R", "owner/repo", "--ref", "master", "--file", ".workflow/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `Note: using file "ci.yml" (input: ".workflow/ci.yml")`) {
		t.Errorf("expected filename note, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotBody, `"fileName":"ci.yml"`) {
		t.Errorf("expected normalized fileName in body, got %s", gotBody)
	}
	if strings.Contains(gotBody, `.workflow/ci.yml`) {
		t.Errorf("expected no path prefix in body, got %s", gotBody)
	}
}

// TestPipelineRunFileNameBaseNotSent verifies a bare --file (no path) stays
// as-is and no note line is printed (the round-trip is unchanged).
func TestPipelineRunFileNameBaseNotSent(t *testing.T) {
	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{
			ID:          7,
			BuildNumber: 12,
			Status:      "WAITTING",
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "run", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Note: using file") {
		t.Errorf("expected no note for bare filename, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotBody, `"fileName":"ci.yml"`) {
		t.Errorf("expected bare fileName in body, got %s", gotBody)
	}
}

// TestPipelineRunInvalidParam asserts a malformed KEY=VALUE is rejected and that
// the trigger (post-billing) call never happens.
func TestPipelineRunInvalidParam(t *testing.T) {
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
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{ID: 7, BuildNumber: 12}))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "run", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml", "-p", "NOVALUE")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `invalid param "NOVALUE", expected KEY=VALUE`) {
		t.Errorf("expected invalid param error, got %v", err)
	}
	// billing may run first, but the trigger must never fire.
	mu.Lock()
	defer mu.Unlock()
	for _, p := range gotPaths {
		if strings.HasSuffix(p, "/pipelines/builds") {
			t.Errorf("expected no trigger request, got %v", gotPaths)
		}
	}
}

// TestPipelineRunNilBuild asserts a nil trigger response surfaces an error.
func TestPipelineRunNilBuild(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "run", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml")
	if err == nil || !strings.Contains(err.Error(), "trigger build returned no build") {
		t.Errorf("expected nil build error, got %v", err)
	}
}

// TestPipelineRunMissingFlags asserts cobra's required-flag validation fires.
func TestPipelineRunMissingFlags(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil)
	_, _, err := runPipelineCmd(t, f, "run", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "file", "ref" not set`) {
		t.Errorf("expected required flags error, got %v", err)
	}
}
