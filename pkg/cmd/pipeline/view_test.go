package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// TestPipelineView asserts the view command renders File/Ref/Name/YAML and
// performs the repo-scoped billing check first.
func TestPipelineView(t *testing.T) {
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
		_, _ = w.Write(jsonBody(t, giteego.PipelineYamlSummaryVO{
			FileName: "ci.yml",
			Name:     "CI",
			Ref:      "master",
			Yaml:     "stages:\n- build",
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "view", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "File:     ci.yml") {
		t.Errorf("expected File line, got:\n%s", out)
	}
	if !strings.Contains(out, "Ref:      master") {
		t.Errorf("expected Ref line, got:\n%s", out)
	}
	if !strings.Contains(out, "Name:     CI") {
		t.Errorf("expected Name line, got:\n%s", out)
	}
	if !strings.Contains(out, "YAML config") && !strings.Contains(out, "YAML 配置") {
		t.Errorf("expected yaml config header, got:\n%s", out)
	}
	if !strings.Contains(out, "--------") {
		t.Errorf("expected section divider, got:\n%s", out)
	}
	if !strings.Contains(out, "stages:") {
		t.Errorf("expected yaml body, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+view), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines/detail") {
		t.Errorf("expected detail path, got %s", gotPaths[1])
	}
}

// TestPipelineViewNotifyParseError asserts a parse error message is rendered.
func TestPipelineViewNotifyParseError(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PipelineYamlSummaryVO{
			FileName: "ci.yml",
			Ref:      "master",
			Message:  "jobs must have an id",
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "view", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "parse error: jobs must have an id") {
		t.Errorf("expected parse error line, got:\n%s", out)
	}
}

// TestPipelineViewNotFound asserts a nil response surfaces the not-found error.
func TestPipelineViewNotFound(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "view", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml")
	if err == nil || !strings.Contains(err.Error(), `pipeline "ci.yml" not found for ref "master"`) {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestPipelineViewMissingFlags asserts cobra's required-flag validation fires.
func TestPipelineViewMissingFlags(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil)
	_, _, err := runPipelineCmd(t, f, "view", "-R", "owner/repo")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "file", "ref" not set`) {
		t.Errorf("expected required flags error, got %v", err)
	}
}
