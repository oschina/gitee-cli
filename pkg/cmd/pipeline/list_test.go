package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// TestPipelineListTable asserts the table output with a parse-error note.
func TestPipelineListTable(t *testing.T) {
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
		_, _ = w.Write(jsonBody(t, []giteego.PipelineYamlSummaryVO{
			{FileName: "ci.yml", Name: "CI"},
			{FileName: "broken.yml", Name: "Broken", Message: "syntax is invalid"},
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "list", "-R", "owner/repo", "--ref", "master")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "FILE") || !strings.Contains(out, "NAME") || !strings.Contains(out, "NOTE") {
		t.Errorf("expected table headers, got:\n%s", out)
	}
	if !strings.Contains(out, "ci.yml") || !strings.Contains(out, "CI") {
		t.Errorf("expected row 1, got:\n%s", out)
	}
	if !strings.Contains(out, "broken.yml") || !strings.Contains(out, "parse error: syntax is invalid") {
		t.Errorf("expected parse-error note, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+list), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines") {
		t.Errorf("expected pipelines path, got %s", gotPaths[1])
	}
}

// TestPipelineListEmpty asserts the no-files message for an empty ref.
func TestPipelineListEmpty(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, []giteego.PipelineYamlSummaryVO{}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "list", "-R", "owner/repo", "--ref", "develop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No pipeline YAML files found for ref develop") &&
		!strings.Contains(out, "在 ref develop 下没有找到流水线 YAML 文件") {
		t.Errorf("expected empty message, got:\n%s", out)
	}
}

// TestPipelineListMissingRef asserts cobra's required-flag validation fires.
func TestPipelineListMissingRef(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil)
	_, _, err := runPipelineCmd(t, f, "list", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "ref" not set`) {
		t.Errorf("expected required flag error, got %v", err)
	}
}
