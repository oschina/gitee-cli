package pipeline

import (
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// TestPipelineListNotEnabledErrors verifies that when gitee-go is not enabled
// for the repo, the command fails with the enable-page URL and tells the user
// to enable it themselves. There is no auto-open logic and no browser is
// launched.
func TestPipelineListNotEnabledErrors(t *testing.T) {
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			// gitee-go not open for this repo.
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"gitee_go_status": "not_open"}))
			return
		}
		// The pipeline list query must never be reached.
		_, _ = w.Write(jsonBody(t, []giteego.PipelineYamlSummaryVO{{FileName: "ci.yml"}}))
	}, &gotPathBase)

	out, errOut, err := runPipelineCmd(t, f, "list", "-R", "owner/repo", "--ref", "master")
	if err == nil {
		t.Fatalf("expected not-enabled error, got nil\nerrOut=%s", errOut)
	}
	// The error surfaces the enable-page URL so the user can open it manually.
	if !strings.Contains(err.Error(), "/owner/repo/gitee_go/open?qt=path") {
		t.Errorf("expected enable URL in error, got: %v", err)
	}
	if strings.Contains(out, "ci.yml") {
		t.Errorf("expected no pipeline list output when not enabled, got:\n%s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
}
