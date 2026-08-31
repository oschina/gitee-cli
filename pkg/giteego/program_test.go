package giteego

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestProgramListQueryToQuery verifies zero values are dropped so the backend
// uses its own defaults, and that slices are comma-joined.
func TestProgramListQueryToQuery(t *testing.T) {
	tests := []struct {
		name string
		q    ProgramListQuery
		want map[string]string
	}{
		{name: "empty", q: ProgramListQuery{}, want: map[string]string{}},
		{
			name: "paging", q: ProgramListQuery{Current: 2, PageSize: 50},
			want: map[string]string{"current": "2", "pageSize": "50"},
		},
		{
			name: "statuses joined",
			q:    ProgramListQuery{Statuses: []string{"SUCCESS", "FAILED"}},
			want: map[string]string{"statuses": "SUCCESS,FAILED"},
		},
		{
			name: "label ids joined",
			q:    ProgramListQuery{LabelIDs: []int64{3, 7, 9}},
			want: map[string]string{"labelIds": "3,7,9"},
		},
		{
			name: "full",
			q: ProgramListQuery{
				Current: 1, PageSize: 20, Statuses: []string{"RUNNING"},
				LabelIDs: []int64{5}, GroupID: 4,
				Order: "create_time", Sort: "desc", Search: "deploy",
			},
			want: map[string]string{
				"current": "1", "pageSize": "20", "statuses": "RUNNING",
				"labelIds": "5", "groupId": "4", "order": "create_time",
				"sort": "desc", "search": "deploy",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.q.toQuery()
			if len(got) != len(tt.want) {
				t.Fatalf("toQuery() = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("toQuery()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// reqInfo captures a single request observed by the in-memory transport.
type reqInfo struct {
	method string
	path   string
	query  url.Values
	body   []byte
}

// runAndRecord drives run through a testClient and returns the last recorded
// request (method, path, query, body).
func runAndRecord(t *testing.T, token string, handler http.HandlerFunc, run func(*Client) error) reqInfo {
	t.Helper()
	c, rt := testClient(token, handler)
	if err := run(c); err != nil {
		t.Fatalf("run: %v", err)
	}
	reqs := rt.requests()
	if len(reqs) == 0 {
		t.Fatalf("no request recorded")
	}
	r := reqs[len(reqs)-1]
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	return reqInfo{method: r.Method, path: r.URL.Path, query: r.URL.Query(), body: body}
}

// TestProgramPipelineCRUD covers list/get/create/update/clone/delete paths and
// verbs for program pipelines.
func TestProgramPipelineCRUD(t *testing.T) {
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantBody   any
		resp       any
		run        func(*Client) error
	}{
		{
			name: "list", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines",
			resp: map[string]any{"current": 1, "pageSize": 10, "total": 0, "data": []any{}},
			run: func(c *Client) error {
				_, err := c.ListProgramPipelines(context.Background(), ProgramListQuery{})
				return err
			},
		},
		{
			name: "get", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/12",
			resp: map[string]any{"id": 12, "name": "pl"},
			run:  func(c *Client) error { _, err := c.GetProgramPipeline(context.Background(), 12); return err },
		},
		{
			name: "create", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines",
			wantBody: PipelineRequest{Name: "new"},
			resp:     map[string]any{"id": 12, "name": "new"},
			run: func(c *Client) error {
				_, err := c.CreateProgramPipeline(context.Background(), PipelineRequest{Name: "new"})
				return err
			},
		},
		{
			name: "update", wantMethod: http.MethodPut, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/12",
			wantBody: PipelineRequest{Name: "renamed"},
			resp:     map[string]any{"id": 12, "name": "renamed"},
			run: func(c *Client) error {
				_, err := c.UpdateProgramPipeline(context.Background(), 12, PipelineRequest{Name: "renamed"})
				return err
			},
		},
		{
			name: "clone", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/12/clone",
			resp: map[string]any{"id": 13, "name": "clone"},
			run:  func(c *Client) error { _, err := c.CloneProgramPipeline(context.Background(), 12); return err },
		},
		{
			name: "delete", wantMethod: http.MethodDelete, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/12",
			resp: true,
			run:  func(c *Client) error { return c.DeleteProgramPipeline(context.Background(), 12) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, jsonBody(t, tt.resp))
			}, tt.run)
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

// TestProgramListPipelineHistoryQuery asserts the history route and that pageSize
// zero is dropped (backend default).
func TestProgramListPipelineHistoryQuery(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{"current": 1, "pageSize": 10}))
	}, func(c *Client) error {
		_, err := c.ListProgramPipelineHistory(context.Background(), 12, 1, 0)
		return err
	})
	if !strings.Contains(rr.path, "/rest/v5/multi-source/pipelines/12/history") {
		t.Errorf("path = %q, want history route", rr.path)
	}
	if rr.query.Get("current") != "1" {
		t.Errorf("current = %q, want 1", rr.query.Get("current"))
	}
	if rr.query.Get("pageSize") != "" {
		t.Errorf("pageSize = %q, want dropped", rr.query.Get("pageSize"))
	}
}

// TestGetProgramPipelineHistoryPath asserts the history/{id} route.
func TestGetProgramPipelineHistoryPath(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{"id": 5, "version": 2}))
	}, func(c *Client) error {
		_, err := c.GetProgramPipelineHistory(context.Background(), 5)
		return err
	})
	want := "/gitee-go/ipipe/rest/v5/multi-source/pipelines/history/5"
	if rr.method != http.MethodGet || rr.path != want {
		t.Errorf("got %s %s, want GET %s", rr.method, rr.path, want)
	}
}

// TestApplyProgramPipelineHistoryPath asserts POST history/{id}/apply.
func TestApplyProgramPipelineHistoryPath(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{"id": 12, "name": "pl"}))
	}, func(c *Client) error {
		_, err := c.ApplyProgramPipelineHistory(context.Background(), 5)
		return err
	})
	want := "/gitee-go/ipipe/rest/v5/multi-source/pipelines/history/5/apply"
	if rr.method != http.MethodPost || rr.path != want {
		t.Errorf("got %s %s, want POST %s", rr.method, rr.path, want)
	}
}

// TestTriggerProgramBuild asserts the trigger body (pipelineId + params) and the
// builds route.
func TestTriggerProgramBuild(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{"id": 9001, "buildNumber": 42, "status": "WAITTING"}))
	}, func(c *Client) error {
		_, err := c.TriggerProgramBuild(context.Background(), PipelineOpsBuildRequest{
			PipelineID: 706,
			Params:     []ParameterStruct{{Key: "GITEE_BRANCH", DefaultValue: "master"}},
		})
		return err
	})
	want := "/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds"
	if rr.method != http.MethodPost || rr.path != want {
		t.Errorf("got %s %s, want POST %s", rr.method, rr.path, want)
	}
	var req PipelineOpsBuildRequest
	if err := json.Unmarshal(rr.body, &req); err != nil {
		t.Fatalf("decode body: %v (%s)", err, rr.body)
	}
	if req.PipelineID != 706 || len(req.Params) != 1 || req.Params[0].Key != "GITEE_BRANCH" {
		t.Errorf("body = %s, want pipelineId 706 with GITEE_BRANCH", rr.body)
	}
}

// TestProgramBuildIdentifierQuery asserts build history is keyed by the bare
// numeric pipeline id (not the "pipeline.ops.pipeline.{id}" form, which the
// backend treats as an unknown pipeline and returns an empty page).
func TestProgramBuildIdentifierQuery(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{"current": 1, "pageSize": 20}))
	}, func(c *Client) error {
		_, err := c.ListProgramBuildHistory(context.Background(), 706, ProgramListQuery{Current: 1, PageSize: 20})
		return err
	})
	if got := rr.query.Get("identifier"); got != "706" {
		t.Errorf("identifier = %q, want 706", got)
	}
	if rr.query.Get("current") != "1" || rr.query.Get("pageSize") != "20" {
		t.Errorf("query = %v, want current=1 pageSize=20", rr.query)
	}
}

// TestProgramLastBuildIdentifierQuery asserts builds/last carries the bare
// numeric identifier too.
func TestProgramLastBuildIdentifierQuery(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{"id": 9001}))
	}, func(c *Client) error {
		_, err := c.GetProgramLastBuild(context.Background(), 706)
		return err
	})
	if got := rr.query.Get("identifier"); got != "706" {
		t.Errorf("identifier = %q, want 706", got)
	}
}

// TestProgramBuildOps covers build view/status and the side-effect ops
// (cancel/rebuild, stage, job).
func TestProgramBuildOps(t *testing.T) {
	buildData := func(id int64) map[string]any {
		return map[string]any{"id": id, "buildNumber": 42, "status": "WAITTING"}
	}
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		resp       any
		run        func(*Client) error
	}{
		{name: "get build", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001", resp: buildData(9001), run: func(c *Client) error { _, err := c.GetProgramBuild(context.Background(), 9001); return err }},
		{name: "build status", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001/status", resp: map[string]any{"id": 9001, "status": "RUNNING"}, run: func(c *Client) error { _, err := c.GetProgramBuildStatus(context.Background(), 9001); return err }},
		{name: "cancel build", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001/cancel", resp: "", run: func(c *Client) error { return c.CancelProgramBuild(context.Background(), 9001) }},
		{name: "rebuild build", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/9001/rebuild", resp: buildData(9001), run: func(c *Client) error { _, err := c.RebuildProgramBuild(context.Background(), 9001); return err }},
		{name: "get stage", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/builds/9002", resp: buildData(9002), run: func(c *Client) error { _, err := c.GetProgramStageBuild(context.Background(), 9002); return err }},
		{name: "cancel stage", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/builds/9002/cancel", resp: "", run: func(c *Client) error { return c.CancelProgramStageBuild(context.Background(), 9002) }},
		{name: "retry stage", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/builds/9002/retry", resp: "", run: func(c *Client) error { return c.RetryProgramStageBuild(context.Background(), 9002) }},
		{name: "continue stage", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/builds/9002/continue", resp: "", run: func(c *Client) error { return c.ContinueProgramStageBuild(context.Background(), 9002) }},
		{name: "cancel job", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/jobs/builds/9003/cancel", resp: "", run: func(c *Client) error { return c.CancelProgramJobBuild(context.Background(), 9003) }},
		{name: "skip job", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/jobs/builds/9003/skip", resp: "", run: func(c *Client) error { return c.SkipProgramJobBuild(context.Background(), 9003) }},
		{name: "retry job", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/jobs/builds/9003/retry", resp: "", run: func(c *Client) error { return c.RetryProgramJobBuild(context.Background(), 9003) }},
		{name: "mark job success", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/stages/jobs/builds/9003/mark-as-success", resp: "", run: func(c *Client) error {
			return c.MarkProgramJobAsSuccess(context.Background(), 9003, "verified manually")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, jsonBody(t, tt.resp))
			}, tt.run)
			if rr.method != tt.wantMethod {
				t.Errorf("method = %s, want %s", rr.method, tt.wantMethod)
			}
			if rr.path != tt.wantPath {
				t.Errorf("path = %q, want %q", rr.path, tt.wantPath)
			}
		})
	}
}

// TestMarkProgramJobAsSuccessBody asserts the reason body is sent.
func TestMarkProgramJobAsSuccessBody(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, ""))
	}, func(c *Client) error {
		return c.MarkProgramJobAsSuccess(context.Background(), 9003, "no explaination")
	})
	var req JobMarkSuccessRequest
	if err := json.Unmarshal(rr.body, &req); err != nil {
		t.Fatalf("decode body: %v (%s)", err, rr.body)
	}
	if req.Reason != "no explaination" {
		t.Errorf("reason = %q, want sent value", req.Reason)
	}
}

// TestProgramParamCRUD covers list/get/create/update/clone/delete for program
// parameter templates, asserting the clone verb is PUT.
func TestProgramParamCRUD(t *testing.T) {
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantBody   any
		resp       any
		run        func(*Client) error
	}{
		{
			name: "list", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/params",
			resp: []any{map[string]any{"id": 12, "name": "p"}},
			run:  func(c *Client) error { _, err := c.ListProgramParams(context.Background()); return err },
		},
		{
			name: "get", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/params/12",
			resp: map[string]any{"id": 12, "name": "p"},
			run:  func(c *Client) error { _, err := c.GetProgramParam(context.Background(), 12); return err },
		},
		{
			name: "create", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/params",
			wantBody: ParamRequest{Name: "build-params"},
			resp:     map[string]any{"id": 12, "name": "build-params"},
			run: func(c *Client) error {
				_, err := c.CreateProgramParam(context.Background(), ParamRequest{Name: "build-params"})
				return err
			},
		},
		{
			name: "update", wantMethod: http.MethodPut, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/params/12",
			wantBody: ParamRequest{Name: "renamed"},
			resp:     map[string]any{"id": 12, "name": "renamed"},
			run: func(c *Client) error {
				_, err := c.UpdateProgramParam(context.Background(), 12, ParamRequest{Name: "renamed"})
				return err
			},
		},
		{
			name: "clone", wantMethod: http.MethodPut, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/params/12/clone",
			resp: map[string]any{"id": 13, "name": "clone"},
			run:  func(c *Client) error { _, err := c.CloneProgramParam(context.Background(), 12); return err },
		},
		{
			name: "delete", wantMethod: http.MethodDelete, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/params/12",
			resp: true,
			run:  func(c *Client) error { return c.DeleteProgramParam(context.Background(), 12) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, jsonBody(t, tt.resp))
			}, tt.run)
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

// TestProgramTemplateCRUD covers list/get/create/update/delete/enable/disable and
// categories for program pipeline templates.
func TestProgramTemplateCRUD(t *testing.T) {
	templateData := func(id int64) map[string]any { return map[string]any{"id": id, "name": "deploy"} }
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantBody   any
		resp       any
		run        func(*Client) error
	}{
		{
			name: "list", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates",
			resp: []any{templateData(12)},
			run:  func(c *Client) error { _, err := c.ListProgramTemplates(context.Background()); return err },
		},
		{
			name: "get", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12",
			resp: templateData(12),
			run:  func(c *Client) error { _, err := c.GetProgramTemplate(context.Background(), 12); return err },
		},
		{
			name: "create", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates",
			wantBody: ProgramPipelineTemplateRequest{Name: "deploy"},
			resp:     templateData(12),
			run: func(c *Client) error {
				_, err := c.CreateProgramTemplate(context.Background(), ProgramPipelineTemplateRequest{Name: "deploy"})
				return err
			},
		},
		{
			name: "update", wantMethod: http.MethodPut, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12",
			wantBody: ProgramPipelineTemplateRequest{Name: "renamed"},
			resp:     templateData(12),
			run: func(c *Client) error {
				_, err := c.UpdateProgramTemplate(context.Background(), 12, ProgramPipelineTemplateRequest{Name: "renamed"})
				return err
			},
		},
		{
			name: "delete", wantMethod: http.MethodDelete, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12",
			resp: true,
			run:  func(c *Client) error { return c.DeleteProgramTemplate(context.Background(), 12) },
		},
		{
			name: "disable", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12/disable",
			resp: true,
			run:  func(c *Client) error { return c.DisableProgramTemplate(context.Background(), 12) },
		},
		{
			name: "enable", wantMethod: http.MethodPost, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12/enable",
			resp: true,
			run:  func(c *Client) error { return c.EnableProgramTemplate(context.Background(), 12) },
		},
		{
			name: "categories", wantMethod: http.MethodGet, wantPath: "/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/categories",
			resp: []any{map[string]any{"id": 1, "name": "部署"}},
			run:  func(c *Client) error { _, err := c.ListProgramTemplateCategories(context.Background()); return err },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, jsonBody(t, tt.resp))
			}, tt.run)
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

// TestProgramPluginExampleYaml asserts the jobType query is passed.
func TestProgramPluginExampleYaml(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, "- step:"))
	}, func(c *Client) error {
		yaml, err := c.GenerateProgramPluginExampleYaml(context.Background(), "MAVEN_JOB")
		if yaml != "- step:" {
			t.Errorf("yaml = %q", yaml)
		}
		return err
	})
	want := "/gitee-go/ipipe/rest/v5/multi-source/plugins/example"
	if rr.method != http.MethodGet || rr.path != want {
		t.Errorf("got %s %s, want GET %s", rr.method, rr.path, want)
	}
	if got := rr.query.Get("jobType"); got != "MAVEN_JOB" {
		t.Errorf("jobType = %q, want MAVEN_JOB", got)
	}
}

// TestProgramListPlugins decodes the grouped plugin payload and asserts the
// project (multi-source) route.
func TestProgramListPlugins(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, []CategoryPluginVO{{
			Name:    "构建",
			Plugins: []PluginVO{{Name: "maven-build", Type: "MAVEN_JOB"}},
		}}))
	}, func(c *Client) error {
		cats, err := c.ListProgramPlugins(context.Background())
		if err != nil {
			return err
		}
		if len(cats) != 1 || len(cats[0].Plugins) != 1 || cats[0].Plugins[0].Type != "MAVEN_JOB" {
			t.Errorf("unexpected categories: %+v", cats)
		}
		return nil
	})
	want := "/gitee-go/ipipe/rest/v5/multi-source/plugins"
	if rr.path != want {
		t.Errorf("path = %q, want %q", rr.path, want)
	}
}

// TestGetProgramPluginSchemesMapForm covers the map-form scheme response and the
// single-scheme fallback on the multi-source route.
func TestGetProgramPluginSchemesMapForm(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{
			"MAVEN_JOB": map[string]any{
				"type":   map[string]any{"json": "MAVEN_JOB"},
				"config": []any{},
			},
		}))
	}, func(c *Client) error {
		schemes, err := c.GetProgramPluginSchemes(context.Background(), "MAVEN_JOB")
		if err != nil {
			return err
		}
		s, ok := schemes["MAVEN_JOB"]
		if !ok || s == nil || s.Type.JSON != "MAVEN_JOB" {
			t.Errorf("schemes = %+v, want MAVEN_JOB entry", schemes)
		}
		return nil
	})
	if !strings.Contains(rr.path, "/rest/v5/multi-source/plugins/scheme") {
		t.Errorf("path = %q, want scheme route", rr.path)
	}
}

func TestGetProgramPluginSchemesSingleFallback(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{
			"type":   map[string]any{"json": "MAVEN_JOB"},
			"config": []any{},
		}))
	})
	schemes, err := c.GetProgramPluginSchemes(context.Background(), "MAVEN_JOB")
	if err != nil {
		t.Fatal(err)
	}
	s, ok := schemes["MAVEN_JOB"]
	if !ok || s == nil {
		t.Errorf("schemes = %+v, want single-scheme fallback under jobType", schemes)
	}
}

// TestProgramMethodsCodeNonZeroError verifies code!=0 surfaces the msg.
func TestProgramMethodsCodeNonZeroError(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":404,"msg":"pipeline not found","data":null}`)
	})
	_, err := c.GetProgramPipeline(context.Background(), 404)
	if err == nil || !strings.Contains(err.Error(), "pipeline not found") {
		t.Errorf("expected code!=0 error, got %v", err)
	}
}
