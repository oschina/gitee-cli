package pipeline

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// writeTempYaml writes content to a temp file inside t.TempDir and returns its
// path.
func writeTempYaml(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// commitHandler returns a handler that serves the billing status check plus the
// gitee-code yaml commit endpoint, recording the commit request.
func commitHandler(t *testing.T, got *[]string, gotPathBase *string) http.HandlerFunc {
	t.Helper()
	var mu sync.Mutex
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		*got = append(*got, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, "success"))
	}
}

// TestPipelineCommit verifies the commit command: billing check first, then
// POST to the gitee-code yaml commits route with the local YAML content, and a
// readable success output. A path-style --file is reduced to its bare name —
// the backend fileName never carries a directory prefix.
func TestPipelineCommit(t *testing.T) {
	yamlContent := "version: \"1.0\"\nname: ci\nstages: []\n"
	yamlPath := writeTempYaml(t, "ci.yml", yamlContent)

	var mu sync.Mutex
	var gotPaths []string
	var gotBody string
	var gotPathBase string
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
		_, _ = w.Write(jsonBody(t, "success"))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "commit",
		"-R", "owner/repo",
		"--ref", "master",
		"--file", ".gitee/workflows/ci.yml",
		"--yaml", yamlPath,
		"-m", "feat: add CI pipeline",
		"--yes")
	if err != nil {
		t.Fatal(err)
	}
	// The success banner is localized; assert the locale-independent aligned
	// block instead.
	if !strings.Contains(out, "Repo:    owner/repo") {
		t.Errorf("expected repo line in output: %s", out)
	}
	if !strings.Contains(out, "Ref:     master") || !strings.Contains(out, "File:    ci.yml") {
		t.Errorf("expected ref/file lines in output: %s", out)
	}
	if !strings.Contains(out, "Message: feat: add CI pipeline") {
		t.Errorf("expected message line in output: %s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+commit), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "billing/gitee-go-service/status") {
		t.Errorf("expected billing check first, got %s", gotPaths[0])
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/external/sources/gitee-code/yaml/commits") {
		t.Errorf("expected yaml commit path with repo path, got %s", gotPaths[1])
	}
	var req giteego.GiteeCodeCommitYamlRequest
	if err := json.Unmarshal([]byte(gotBody), &req); err != nil {
		t.Fatalf("parse commit body %q: %v", gotBody, err)
	}
	if req.Branch != "master" || req.FileName != "ci.yml" {
		t.Errorf("branch/file = %q/%q, want master/ci.yml (bare name, no path)", req.Branch, req.FileName)
	}
	if req.Content != yamlContent {
		t.Errorf("content = %q, want %q", req.Content, yamlContent)
	}
	if req.CommitMessage != "feat: add CI pipeline" {
		t.Errorf("commit message = %q", req.CommitMessage)
	}
}

// TestPipelineCommitFileNameDefault verifies --file may be omitted: the
// repository-side fileName defaults to the local yaml file's base name (no
// directory prefix).
func TestPipelineCommitFileNameDefault(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	yamlPath := filepath.Join(sub, "流水线.yml")
	if err := os.WriteFile(yamlPath, []byte("stages: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		gotBody = readBody(t, r)
		_, _ = w.Write(jsonBody(t, "success"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "commit",
		"-R", "owner/repo", "--ref", "master", "--yaml", yamlPath, "--yes")
	if err != nil {
		t.Fatal(err)
	}
	var req giteego.GiteeCodeCommitYamlRequest
	if err := json.Unmarshal([]byte(gotBody), &req); err != nil {
		t.Fatalf("parse commit body %q: %v", gotBody, err)
	}
	if req.FileName != "流水线.yml" {
		t.Errorf("fileName = %q, want 流水线.yml (base name, no path)", req.FileName)
	}
}

// TestPipelineCommitStdinRequiresFileName verifies stdin input needs an
// explicit --file (stdin has no name to derive one from).
func TestPipelineCommitStdinRequiresFileName(t *testing.T) {
	f := newTestFactory(t, commitHandler(t, &[]string{}, nil), nil)
	f.IOStreams.In = io.NopCloser(strings.NewReader("stages: []\n"))

	_, _, err := runPipelineCmd(t, f, "commit",
		"-R", "owner/repo", "--ref", "master", "--yaml", "-", "--yes")
	if err == nil || !strings.Contains(err.Error(), "fileName is required") {
		t.Errorf("expected fileName required error, got %v", err)
	}
}

// TestPipelineCommitDefaultMessage verifies a sensible commit message is used
// when --message is omitted.
func TestPipelineCommitDefaultMessage(t *testing.T) {
	yamlPath := writeTempYaml(t, "ci.yml", "stages: []\n")

	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		gotBody = readBody(t, r)
		_, _ = w.Write(jsonBody(t, "success"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "commit",
		"-R", "owner/repo", "--ref", "master", "--file", "ci.yml",
		"--yaml", yamlPath, "--yes")
	if err != nil {
		t.Fatal(err)
	}
	var req giteego.GiteeCodeCommitYamlRequest
	if err := json.Unmarshal([]byte(gotBody), &req); err != nil {
		t.Fatalf("parse commit body %q: %v", gotBody, err)
	}
	if req.CommitMessage != "feat: update pipeline configuration" {
		t.Errorf("default commit message = %q", req.CommitMessage)
	}
}

// TestPipelineCommitRequiresYes verifies the side-effect confirmation gate:
// non-interactive runs must pass --yes.
func TestPipelineCommitRequiresYes(t *testing.T) {
	yamlPath := writeTempYaml(t, "ci.yml", "stages: []\n")
	f := newTestFactory(t, commitHandler(t, &[]string{}, nil), nil)

	_, _, err := runPipelineCmd(t, f, "commit",
		"-R", "owner/repo", "--ref", "master", "--file", "ci.yml",
		"--yaml", yamlPath)
	if err == nil || !strings.Contains(err.Error(), "--yes is required in non-interactive mode") {
		t.Errorf("expected --yes gate error, got %v", err)
	}
}

// TestPipelineCommitEmptyYaml verifies an empty YAML file is rejected.
func TestPipelineCommitEmptyYaml(t *testing.T) {
	yamlPath := writeTempYaml(t, "empty.yml", "   \n")
	f := newTestFactory(t, commitHandler(t, &[]string{}, nil), nil)

	_, _, err := runPipelineCmd(t, f, "commit",
		"-R", "owner/repo", "--ref", "master", "--file", "ci.yml",
		"--yaml", yamlPath, "--yes")
	if err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Errorf("expected empty yaml error, got %v", err)
	}
}
