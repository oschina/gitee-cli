package pipeline

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// The program pipeline commands locate the gitee-go gateway by -E <enterprise>
// and -P <program> and route through the /multi-source segment. Unlike the repo
// pipeline commands they do NOT perform a billing open-check: the program (DB
// scope) handlers respond directly with their data.

// programSummary returns a program pipeline summary fixture whose numeric id
// is the bare pipelineOps identifier (the list endpoint returns the id directly,
// e.g. "710").
func programSummary(id int64, name string) giteego.PipelineSummaryVO {
	return giteego.PipelineSummaryVO{
		Identifier: idString(id),
		Name:       name,
	}
}

func idString(id int64) string {
	return strings.TrimSpace(jsonNumber(id))
}

func jsonNumber(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}

func TestProgramPipelineList(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineSummaryVO]{
			Current:  1,
			PageSize: 20,
			Total:    2,
			Data: []giteego.PipelineSummaryVO{
				programSummary(706, "build-all"),
				programSummary(707, "deploy"),
			},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "program", "list", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if gotPathBase != "2/423" {
		t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
	}
	if !strings.Contains(out, "Total: 2") {
		t.Errorf("expected total, got: %s", out)
	}
	// The gateway URL must carry ent/program before /gitee-go and the
	// /multi-source segment.
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing check), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines") {
		t.Errorf("expected program pipelines path with ent/program and multi-source, got %s", gotPaths[0])
	}
	// The table renders the id from the bare numeric identifier.
	for _, want := range []string{"ID", "PIPELINE", "GROUP", "STATUS", "BUILD", "START", "DURATION", "TRIGGER", "UUID"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected header %q in output, got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "│ 706 │ build-all │") || !strings.Contains(out, "│ 707 │ deploy") {
		t.Errorf("expected id+name rows, got:\n%s", out)
	}
}

func TestProgramPipelineListEmpty(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineSummaryVO]{}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "list", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, i18n.T("pipeline.program.no_pipelines")) {
		t.Errorf("expected %q, got: %s", i18n.T("pipeline.program.no_pipelines"), out)
	}
}

func TestProgramPipelineListRequiresLocation(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}, nil), "program", "list")
	if err == nil || !strings.Contains(err.Error(), "enterprise id is required: use --enterprise/-E <id>") {
		t.Errorf("expected enterprise error, got %v", err)
	}

	_, _, err = runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}, nil), "program", "list", "-E", "2")
	if err == nil || !strings.Contains(err.Error(), "program id is required: use --program/-P <id>") {
		t.Errorf("expected program error, got %v", err)
	}
}

func TestProgramPipelineRun(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	var gotBody []byte
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, map[string]any{
			"id": 9001, "buildNumber": 42, "status": "WAITTING",
		}))
	}, nil)

	out, errOut, err := runPipelineCmd(t, f, "program", "run", "--pipeline", "706", "-E", "2", "-P", "423", "-p", "GITEE_BRANCH=master", "-p", "FOO=bar")
	if err != nil {
		t.Fatalf("%v\nout: %s\nerr: %s", err, out, errOut)
	}
	if !strings.Contains(out, "Build #42 triggered") || !strings.Contains(out, "ID:     9001") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "GITEE_BRANCH=master") || !strings.Contains(out, "FOO=bar") {
		t.Errorf("expected params echoed, got: %s", out)
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotPath, "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds") {
		t.Errorf("expected run on multi-source pipelines/builds, got %s", gotPath)
	}
	var req giteego.PipelineOpsBuildRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatalf("decode request body: %v (%s)", err, gotBody)
	}
	if req.PipelineID != 706 {
		t.Errorf("expected pipelineId 706, got %d", req.PipelineID)
	}
	if len(req.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(req.Params))
	}
	if req.Params[0].Key != "GITEE_BRANCH" || req.Params[0].DefaultValue != "master" {
		t.Errorf("unexpected first param: %+v", req.Params[0])
	}
}

func TestProgramPipelineRunInvalidParam(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}, nil), "program", "run", "--pipeline", "706", "-E", "2", "-P", "423", "-p", "NOEQUALS")
	if err == nil || !strings.Contains(err.Error(), `invalid param "NOEQUALS", expected KEY=VALUE`) {
		t.Errorf("expected invalid param error, got %v", err)
	}
}

func TestProgramPipelineRunRequiresPipeline(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}, nil), "program", "run", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "pipeline" not set`) {
		t.Errorf("expected --pipeline required-flag error, got %v", err)
	}
}

func TestProgramBuildList(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineBuildSimpleVO]{
			Current:  1,
			PageSize: 20,
			Total:    1,
			Data: []giteego.PipelineBuildSimpleVO{{
				ID:               9001,
				BelongType:       "PIPELINE_OPS",
				BelongIdentifier: "pipeline.ops.pipeline.706",
				BuildNumber:      42,
				StartTime:        mustTime(t, "2026-08-17 10:00:00"),
				Status:           "SUCCESS",
				Ref:              "master",
				Sources: []giteego.BuildSourceVO{{
					Name: "git", Type: "GIT",
					Source: &giteego.BuildSourceDetail{
						Branch:            "master",
						Event:             "push_hooks",
						Message:           "chore: bump version",
						PathWithNamespace: "2/423",
					},
				}},
			}},
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "build", "list", "--pipeline", "706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "9001") || !strings.Contains(out, "SUCCESS") {
		t.Errorf("expected build row, got: %s", out)
	}
	if !strings.Contains(out, "master") || !strings.Contains(out, "chore: bump version") {
		t.Errorf("expected source columns, got: %s", out)
	}
	if !strings.Contains(out, "page 1/1 (pageSize 20, total 1)") {
		t.Errorf("expected footer, got: %s", out)
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotPath, "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/builds/history") {
		t.Errorf("expected multi-source builds/history path, got %s", gotPath)
	}
}

func TestProgramBuildListEmpty(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineBuildSimpleVO]{}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "build", "list", "--pipeline", "706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, i18n.T("pipeline.no_results")) {
		t.Errorf("expected %q, got: %s", i18n.T("pipeline.no_results"), out)
	}
}

func TestProgramPipelineHistoryEmpty(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineHistoryVO]{}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "history", "706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, i18n.T("pipeline.no_history")) {
		t.Errorf("expected %q, got: %s", i18n.T("pipeline.no_history"), out)
	}
}

func TestProgramParamList(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, []giteego.ParamVO{{
			ID:          12,
			Name:        "build-params",
			Description: "shared build params",
			Parameters: []giteego.ParameterStruct{{
				Key: "GITEE_BRANCH", DefaultValue: "master",
			}},
			UpdateTime: mustTime(t, "2026-08-17 09:00:00"),
		}}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "param", "list", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "build-params") || !strings.Contains(out, "shared build params") {
		t.Errorf("expected param row, got: %s", out)
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotPath, "/2/423/gitee-go/ipipe/rest/v5/multi-source/params") {
		t.Errorf("expected multi-source params path, got %s", gotPath)
	}
}

func TestProgramParamDeleteRequiresYes(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, map[string]any{}))
	}, nil), "program", "param", "delete", "12", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--yes is required in non-interactive mode") {
		t.Errorf("expected confirmation error, got %v", err)
	}
}

func TestProgramParamDeleteYes(t *testing.T) {
	var mu sync.Mutex
	var gotMethod, gotPath string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPath = r.URL.Path
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true)) // DeleteProgramParam decodes ResultVO[bool]
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "param", "delete", "12", "-E", "2", "-P", "423", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Deleted program param 12") {
		t.Errorf("expected deletion message, got: %s", out)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", gotMethod)
	}
	if !strings.Contains(gotPath, "/2/423/gitee-go/ipipe/rest/v5/multi-source/params/12") {
		t.Errorf("expected multi-source params/12 path, got %s", gotPath)
	}
}

func TestProgramTemplateListEmpty(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, []giteego.ProgramPipelineTemplateVO{}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "list", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, i18n.T("pipeline.program.no_templates")) {
		t.Errorf("expected %q, got: %s", i18n.T("pipeline.program.no_templates"), out)
	}
}

func TestProgramTemplateCategoriesEmpty(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, []giteego.PipelineTemplateCategoryVO{}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "categories", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, i18n.T("pipeline.program.no_categories")) {
		t.Errorf("expected %q, got: %s", i18n.T("pipeline.program.no_categories"), out)
	}
}

func TestProgramTemplateCatalog(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "categories"):
			w.Write(jsonBody(t, []giteego.PipelineTemplateCategoryVO{{
				ID: 1, Name: "部署", Icon: "🚀", Sort: 1,
			}}))
		default:
			w.Write(jsonBody(t, []giteego.ProgramPipelineTemplateVO{{
				ID:          12,
				Name:        "deploy-template",
				Description: "build and deploy",
				Category:    &giteego.PipelineTemplateCategoryVO{Name: "部署"},
			}}))
		}
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "list", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "deploy-template") || !strings.Contains(out, "部署") {
		t.Errorf("expected template row with category, got: %s", out)
	}

	out, _, err = runPipelineCmd(t, f, "program", "template", "categories", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "🚀") || !strings.Contains(out, "部署") {
		t.Errorf("expected categories table, got: %s", out)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests, got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates") {
		t.Errorf("expected templates path, got %s", gotPaths[0])
	}
	if !strings.Contains(gotPaths[1], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/categories") {
		t.Errorf("expected categories path, got %s", gotPaths[1])
	}
}

func TestProgramPluginList(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, []giteego.CategoryPluginVO{{
			Name: "构建",
			Plugins: []giteego.PluginVO{{
				Name: "maven-build", Type: "MAVEN_JOB", Description: "Maven 构建",
			}},
		}}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "plugin", "list", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "构建") || !strings.Contains(out, "maven-build") || !strings.Contains(out, "MAVEN_JOB") {
		t.Errorf("expected plugin table, got: %s", out)
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotPath, "/2/423/gitee-go/ipipe/rest/v5/multi-source/plugins") {
		t.Errorf("expected multi-source plugins path, got %s", gotPath)
	}
}

// TestProgramPipelineView renders a full config, asserting the id derivation
// and config params.
func TestProgramPipelineView(t *testing.T) {
	var gotPath string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{
			Identifier: "pipeline.ops.pipeline.706",
			UUID:       "u-1",
			Name:       "build-all",
			Ref:        "master",
			Parameters: []giteego.ParameterStruct{{
				Key: "GITEE_BRANCH", DefaultValue: "master",
			}},
			Stages: []giteego.StageVO{{Name: "build"}, {Name: "deploy"}},
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "view", "706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ID:         706") || !strings.Contains(out, "Name:       build-all") {
		t.Errorf("unexpected output: %s", out)
	}
	if !strings.Contains(out, "GITEE_BRANCH = master") {
		t.Errorf("expected config params, got: %s", out)
	}
	if !strings.Contains(out, "build") || !strings.Contains(out, "deploy") {
		t.Errorf("expected stages, got: %s", out)
	}
	if !strings.Contains(gotPath, "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/706") {
		t.Errorf("expected pipelines/706 path, got %s", gotPath)
	}
}

// TestProgramPipelineViewIdentifierArg accepts the full identifier form too.
func TestProgramPipelineViewIdentifierArg(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "build-all"}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "view", "pipeline.ops.pipeline.706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ID:         706") {
		t.Errorf("expected derived id 706, got: %s", out)
	}
}
