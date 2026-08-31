package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// TestPipelineExample verifies the example command: billing check first, then
// GET /rest/v5/yaml/example, printing the returned YAML text as-is.
func TestPipelineExample(t *testing.T) {
	exampleYaml := "version: \"1.0\"\nname: example-pipeline-uuid\ndisplayName: 示例流水线\nstages:\n  - name: 示例阶段\n"

	var mu sync.Mutex
	var gotPaths []string
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
		_, _ = w.Write(jsonBody(t, exampleYaml))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "example", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "example-pipeline-uuid") || !strings.Contains(out, "示例流水线") {
		t.Errorf("expected example yaml content, got:\n%s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+example), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "billing/gitee-go-service/status") {
		t.Errorf("expected billing check first, got %s", gotPaths[0])
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/yaml/example") {
		t.Errorf("expected yaml example path with repo path, got %s", gotPaths[1])
	}
}
