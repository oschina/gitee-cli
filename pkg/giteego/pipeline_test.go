package giteego

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// pipelineRespH returns a handler that answers every request with the given data
// wrapped in the ResultVO envelope (matching the concrete decode target passed in
// the run function is the caller's responsibility).
func pipelineRespH(t *testing.T, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, data))
	}
}

// TestRepositoryPipelineList asserts the pipelines list route, ref query, and
// decode of the YAML summary slice.
func TestRepositoryPipelineList(t *testing.T) {
	rr := runAndRecord(t, "tok", pipelineRespH(t, []any{
		map[string]any{"fileName": ".gitee-go/pipelines/deploy.yml", "message": "add deploy"},
	}), func(c *Client) error {
		got, err := c.ListRepositoryPipelines(context.Background(), "master")
		if err != nil {
			return err
		}
		if len(got) != 1 || got[0].FileName != ".gitee-go/pipelines/deploy.yml" {
			t.Errorf("pipelines = %+v", got)
		}
		return nil
	})
	if rr.method != http.MethodGet || rr.path != "/gitee-go/ipipe/rest/v5/pipelines" {
		t.Errorf("got %s %s, want GET /gitee-go/ipipe/rest/v5/pipelines", rr.method, rr.path)
	}
	if rr.query.Get("ref") != "master" {
		t.Errorf("ref = %q, want master", rr.query.Get("ref"))
	}
}

// TestRepositoryPipelineDetail asserts the detail route passes ref + fileName.
func TestRepositoryPipelineDetail(t *testing.T) {
	rr := runAndRecord(t, "tok", pipelineRespH(t, map[string]any{
		"fileName": "deploy.yml", "yaml": "stages: []", "ref": "master",
	}), func(c *Client) error {
		got, err := c.GetRepositoryPipeline(context.Background(), "master", "deploy.yml")
		if err != nil {
			return err
		}
		if got == nil || got.Yaml != "stages: []" {
			t.Errorf("detail = %+v, want yaml body", got)
		}
		return nil
	})
	want := "/gitee-go/ipipe/rest/v5/pipelines/detail"
	if rr.method != http.MethodGet || rr.path != want {
		t.Errorf("got %s %s, want GET %s", rr.method, rr.path, want)
	}
	if rr.query.Get("ref") != "master" || rr.query.Get("fileName") != "deploy.yml" {
		t.Errorf("query = %v, want ref=master fileName=deploy.yml", rr.query)
	}
}

// TestListPipelineFileBranches asserts the file-branches route.
func TestListPipelineFileBranches(t *testing.T) {
	rr := runAndRecord(t, "tok", pipelineRespH(t, []any{
		map[string]any{"fileName": "a.yml", "branches": []any{"master", "dev"}},
	}), func(c *Client) error {
		got, err := c.ListPipelineFileBranches(context.Background())
		if err != nil {
			return err
		}
		if len(got) != 1 || len(got[0].Branches) != 2 {
			t.Errorf("file-branches = %+v", got)
		}
		return nil
	})
	want := "/gitee-go/ipipe/rest/v5/pipelines/file-branches"
	if rr.method != http.MethodGet || rr.path != want {
		t.Errorf("got %s %s, want GET %s", rr.method, rr.path, want)
	}
}

// TestRepositoryBuildOps covers trigger, view, status, cancel, rebuild, last
// and the side-effect ops (stage, job) for repo-scoped builds.
func TestRepositoryBuildOps(t *testing.T) {
	buildData := func() map[string]any {
		return map[string]any{"id": 1001, "buildNumber": 7, "status": "WAITTING"}
	}
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantBody   any
		resp       any
		run        func(*Client) error
	}{
		{name: "trigger", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/builds",
			wantBody: BuildRequest{FileName: "deploy.yml", Ref: "master"},
			resp:     buildData(),
			run: func(c *Client) error {
				_, err := c.TriggerBuild(context.Background(), BuildRequest{FileName: "deploy.yml", Ref: "master"})
				return err
			}},
		{name: "get build", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/builds/1001", resp: buildData(),
			run: func(c *Client) error { _, err := c.GetBuild(context.Background(), 1001); return err }},
		{name: "build status", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/builds/1001/status",
			resp: map[string]any{"id": 1001, "status": "RUNNING"},
			run:  func(c *Client) error { _, err := c.GetBuildStatus(context.Background(), 1001); return err }},
		{name: "cancel build", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/builds/1001/cancel", resp: "",
			run: func(c *Client) error { return c.CancelBuild(context.Background(), 1001) }},
		{name: "rebuild build", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/builds/1001/rebuild", resp: buildData(),
			run: func(c *Client) error { _, err := c.RebuildBuild(context.Background(), 1001); return err }},
		{name: "get stage", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/builds/1002", resp: buildData(),
			run: func(c *Client) error { _, err := c.GetStageBuild(context.Background(), 1002); return err }},
		{name: "cancel stage", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/builds/1002/cancel", resp: "",
			run: func(c *Client) error { return c.CancelStageBuild(context.Background(), 1002) }},
		{name: "retry stage", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/builds/1002/retry", resp: "",
			run: func(c *Client) error { return c.RetryStageBuild(context.Background(), 1002) }},
		{name: "continue stage", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/builds/1002/continue", resp: "",
			run: func(c *Client) error { return c.ContinueStageBuild(context.Background(), 1002) }},
		{name: "cancel job", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/jobs/builds/1003/cancel", resp: "",
			run: func(c *Client) error { return c.CancelJobBuild(context.Background(), 1003) }},
		{name: "skip job", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/jobs/builds/1003/skip", resp: "",
			run: func(c *Client) error { return c.SkipJobBuild(context.Background(), 1003) }},
		{name: "retry job", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/jobs/builds/1003/retry", resp: "",
			run: func(c *Client) error { return c.RetryJobBuild(context.Background(), 1003) }},
		{name: "mark job success", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/pipelines/stages/jobs/builds/1003/mark-as-success",
			wantBody: JobMarkSuccessRequest{Reason: "verified"},
			resp:     "",
			run:      func(c *Client) error { return c.MarkJobAsSuccess(context.Background(), 1003, "verified") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := runAndRecord(t, "tok", pipelineRespH(t, tt.resp), tt.run)
			if rr.method != tt.wantMethod {
				t.Errorf("method = %s, want %s", rr.method, tt.wantMethod)
			}
			if rr.path != tt.wantPath {
				t.Errorf("path = %q, want %q", rr.path, tt.wantPath)
			}
			if tt.wantBody != nil {
				want, _ := json.Marshal(tt.wantBody)
				if string(rr.body) != string(want) {
					t.Errorf("body = %s, want %s", rr.body, want)
				}
			}
		})
	}
}

// TestListBuildHistoryQuery asserts the history route and that the statuses
// slice is comma-joined while single values pass through.
func TestListBuildHistoryQuery(t *testing.T) {
	rr := runAndRecord(t, "tok", pipelineRespH(t, map[string]any{
		"current": 1, "pageSize": 20, "total": 1,
		"data": []any{map[string]any{"id": 1001, "buildNumber": 7}},
	}), func(c *Client) error {
		page, err := c.ListBuildHistory(context.Background(), "deploy.yml", "master", "create_time", "desc", 1, 20, []string{"SUCCESS", "FAILED"})
		if err != nil {
			return err
		}
		if page.Total != 1 || len(page.Data) != 1 || page.Data[0].ID != 1001 {
			t.Errorf("page = %+v", page)
		}
		return nil
	})
	want := "/gitee-go/ipipe/rest/v5/pipelines/builds/history"
	if rr.path != want {
		t.Errorf("path = %q, want %q", rr.path, want)
	}
	q := rr.query
	if q.Get("fileName") != "deploy.yml" || q.Get("ref") != "master" {
		t.Errorf("fileName/ref = %q/%q", q.Get("fileName"), q.Get("ref"))
	}
	if q.Get("current") != "1" || q.Get("pageSize") != "20" {
		t.Errorf("paging = %q/%q", q.Get("current"), q.Get("pageSize"))
	}
	if q.Get("order") != "create_time" || q.Get("sort") != "desc" {
		t.Errorf("order/sort = %q/%q", q.Get("order"), q.Get("sort"))
	}
	if q.Get("statuses") != "SUCCESS,FAILED" {
		t.Errorf("statuses = %q, want SUCCESS,FAILED", q.Get("statuses"))
	}
}

// TestListBuildHistoryNoStatuses asserts statuses is omitted when empty.
func TestListBuildHistoryNoStatuses(t *testing.T) {
	rr := runAndRecord(t, "tok", pipelineRespH(t, map[string]any{
		"current": 1, "pageSize": 20, "total": 0, "data": []any{},
	}), func(c *Client) error {
		_, err := c.ListBuildHistory(context.Background(), "", "", "", "", 1, 20, nil)
		return err
	})
	if rr.query.Get("statuses") != "" {
		t.Errorf("statuses = %q, want omitted", rr.query.Get("statuses"))
	}
}

// TestGetLastBuild asserts the builds/last route.
func TestGetLastBuild(t *testing.T) {
	rr := runAndRecord(t, "tok", pipelineRespH(t, map[string]any{"id": 1001, "status": "SUCCEED"}),
		func(c *Client) error {
			got, err := c.GetLastBuild(context.Background(), "deploy.yml", "master")
			if err != nil {
				return err
			}
			if got == nil || got.ID != 1001 {
				t.Errorf("last build = %+v", got)
			}
			return nil
		})
	want := "/gitee-go/ipipe/rest/v5/pipelines/builds/last"
	if rr.method != http.MethodGet || rr.path != want {
		t.Errorf("got %s %s, want GET %s", rr.method, rr.path, want)
	}
	if rr.query.Get("fileName") != "deploy.yml" || rr.query.Get("ref") != "master" {
		t.Errorf("query = %v", rr.query)
	}
}

// TestRepositoryPipelineCodeNonZeroError verifies code!=0 surfaces the msg for a
// repo-scoped method.
func TestRepositoryPipelineCodeNonZeroError(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":404,"msg":"yaml not found","data":null}`)
	})
	_, err := c.GetRepositoryPipeline(context.Background(), "master", "nope.yml")
	if err == nil || !strings.Contains(err.Error(), "yaml not found") {
		t.Errorf("expected code!=0 error, got %v", err)
	}
}

// TestGetPipelineYamlExample asserts the yaml example route and the YAML-text
// string decode.
func TestGetPipelineYamlExample(t *testing.T) {
	example := "version: \"1.0\"\nname: example-pipeline-uuid\nstages:\n  - name: build\n"
	rr := runAndRecord(t, "tok", pipelineRespH(t, example), func(c *Client) error {
		got, err := c.GetPipelineYamlExample(context.Background())
		if err != nil {
			return err
		}
		if got != example {
			t.Errorf("example = %q, want %q", got, example)
		}
		return nil
	})
	want := "/gitee-go/ipipe/rest/v5/yaml/example"
	if rr.method != http.MethodGet || rr.path != want {
		t.Errorf("got %s %s, want GET %s", rr.method, rr.path, want)
	}
}

// TestCommitRepositoryPipelineYaml asserts the gitee-code yaml commit route and
// that the branch/fileName/content/commitMessage body is sent as-is.
func TestCommitRepositoryPipelineYaml(t *testing.T) {
	req := GiteeCodeCommitYamlRequest{
		Branch:        "master",
		FileName:      ".gitee/workflows/ci.yml",
		Content:       "version: \"1.0\"\nname: my-pipeline\n",
		CommitMessage: "feat: update pipeline configuration",
	}
	rr := runAndRecord(t, "tok", pipelineRespH(t, "success"), func(c *Client) error {
		return c.CommitRepositoryPipelineYaml(context.Background(), req)
	})
	want := "/gitee-go/ipipe/rest/v5/external/sources/gitee-code/yaml/commits"
	if rr.method != http.MethodPost || rr.path != want {
		t.Errorf("got %s %s, want POST %s", rr.method, rr.path, want)
	}
	wantBody, _ := json.Marshal(req)
	if string(rr.body) != string(wantBody) {
		t.Errorf("body = %s, want %s", rr.body, wantBody)
	}
}
