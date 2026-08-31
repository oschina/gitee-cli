package pipeline

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// The program `template` commands are project-scoped and do NOT perform the
// billing open-check: each op is exactly 1 request (or 0 when validation /
// confirmation aborts before the client call). view/create/update/delete/
// enable/disable all route via /multi-source/pipelines/templates.

func templateFixture(id int64, name string) *giteego.ProgramPipelineTemplateVO {
	return &giteego.ProgramPipelineTemplateVO{
		ID:          id,
		Name:        name,
		Description: "build and deploy",
		Category:    &giteego.PipelineTemplateCategoryVO{Name: "部署"},
		Config: &giteego.PipelineVO{
			Ref: "master",
			Parameters: []giteego.ParameterStruct{{
				Key: "GITEE_BRANCH", DefaultValue: "master",
			}},
			Stages: []giteego.StageVO{{Name: "build"}, {Name: "deploy"}},
		},
	}
}

// TestProgramTemplateView asserts the view layout (ID/Name/Description/
// Category/Status + Config ref/params/stages) and the multi-source templates
// path with pathBase 2/423.
func TestProgramTemplateView(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, templateFixture(12, "deploy-template")))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "program", "template", "view", "12", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if gotPathBase != "2/423" {
		t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
	}
	for _, want := range []string{
		"ID:          12\n",
		"Name:        deploy-template\n",
		"Description: build and deploy\n",
		"Category:    部署\n",
		"Status:      enabled\n",
		"Ref:         master\n",
		"GITEE_BRANCH = master\n",
		"Stages\n",
		"  build\n",
		"  deploy\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12") {
		t.Errorf("expected program template path, got %s", gotPaths[0])
	}
}

// TestProgramTemplateViewDisabled asserts the disabled status render.
func TestProgramTemplateViewDisabled(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.ProgramPipelineTemplateVO{ID: 12, Name: "old", Disabled: true}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "view", "12", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Status:      disabled") {
		t.Errorf("expected disabled status, got:\n%s", out)
	}
}

// TestProgramTemplateViewNil asserts a nil template surfaces the not-found
// error.
func TestProgramTemplateViewNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "template", "view", "12", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "program template 12 not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestProgramTemplateCreate asserts `template create` posts the request and
// prints the confirmation line carrying the created id/name.
func TestProgramTemplateCreate(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, templateFixture(18, "deploy-template")))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "create", "-E", "2", "-P", "423",
		"--name", "deploy-template", "--description", "build and deploy")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Created program template 18 (deploy-template)\n") {
		t.Errorf("expected created line, got:\n%s", out)
	}
	if !strings.Contains(gotBody, `"name":"deploy-template"`) || !strings.Contains(gotBody, `"description":"build and deploy"`) {
		t.Errorf("expected name+description in body, got %s", gotBody)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates") {
		t.Errorf("expected templates path, got %s", gotPaths[0])
	}
}

// TestProgramTemplateCreateRequiresName asserts the --name gate fires before any
// request.
func TestProgramTemplateCreateRequiresName(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "template", "create", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected --name requirement, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before validation, got %d", reqCount)
	}
}

// TestProgramTemplateCreateConfigFile asserts create --config supplies the
// PipelineVO config in the request body.
func TestProgramTemplateCreateConfigFile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "pipeline.json")
	if err := os.WriteFile(configFile, []byte(`{"ref":"master","parameters":[{"key":"GITEE_BRANCH","defaultValue":"master"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, templateFixture(18, "deploy-template")))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "template", "create", "-E", "2", "-P", "423",
		"--name", "deploy-template", "--config", configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"name":"deploy-template"`) {
		t.Errorf("expected --name in body, got %s", gotBody)
	}
	// The config is embedded as a PipelineVO (identifier is marshaled empty),
	// so assert the payload fields present rather than one contiguous substring.
	for _, want := range []string{`"config":{`, `"ref":"master"`, `"parameters":[{"key":"GITEE_BRANCH","defaultValue":"master"}]`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("expected %q in config payload, got %s", want, gotBody)
		}
	}
}

// TestProgramTemplateUpdateNothingToUpdate asserts the empty-edit guard fires
// before the client call (no flags -> no request).
func TestProgramTemplateUpdateNothingToUpdate(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "template", "edit", "12", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "nothing to edit: pass --name, --description or --config") {
		t.Errorf("expected nothing-to-edit error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request for empty edit, got %d", reqCount)
	}
}

// TestProgramTemplateUpdate asserts `template edit --name` PUTs to
// templates/{id} and prints the confirmation line.
func TestProgramTemplateUpdate(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, templateFixture(12, "new-name")))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "edit", "12", "-E", "2", "-P", "423", "--name", "new-name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Edited program template 12 (new-name)\n") {
		t.Errorf("expected edited line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodPut || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12") {
		t.Errorf("expected PUT templates/12, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramTemplateDeleteRequiresYes asserts the confirmation gate fires
// before the client call in a non-interactive context (0 requests).
func TestProgramTemplateDeleteRequiresYes(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "template", "delete", "12", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--yes is required in non-interactive mode") {
		t.Errorf("expected confirmation error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before confirmation, got %d", reqCount)
	}
}

// TestProgramTemplateDeleteYes asserts the destructive delete with --yes issues
// the DELETE and prints the confirmation line.
func TestProgramTemplateDeleteYes(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true)) // DeleteProgramTemplate decodes ResultVO[bool]
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "delete", "12", "-E", "2", "-P", "423", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Deleted program template 12\n") {
		t.Errorf("expected deleted line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodDelete || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12") {
		t.Errorf("expected DELETE templates/12, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramTemplateEnable asserts enable posts to templates/{id}/enable with
// no confirmation and prints the confirmation line.
func TestProgramTemplateEnable(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "enable", "12", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Enabled program template 12\n") {
		t.Errorf("expected enabled line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12/enable") {
		t.Errorf("expected POST templates/12/enable, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramTemplateDisableRequiresYes asserts disable's confirmation gate
// fires before the client call (0 requests).
func TestProgramTemplateDisableRequiresYes(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "template", "disable", "12", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--yes is required in non-interactive mode") {
		t.Errorf("expected confirmation error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before confirmation, got %d", reqCount)
	}
}

// TestProgramTemplateDisableYes asserts disable with --yes posts to
// templates/{id}/disable and prints the confirmation line.
func TestProgramTemplateDisableYes(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "template", "disable", "12", "-E", "2", "-P", "423", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Disabled program template 12\n") {
		t.Errorf("expected disabled line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/templates/12/disable") {
		t.Errorf("expected POST templates/12/disable, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramTemplateMissingArg asserts cobra's arity validation fires first.
func TestProgramTemplateMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "program", "template", "view", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}

// TestProgramTemplateIDArg covers the shared programTemplateIDArg pure parser
// branches.
func TestProgramTemplateIDArg(t *testing.T) {
	if _, err := programTemplateIDArg(nil); err == nil || !strings.Contains(err.Error(), "template id is required") {
		t.Errorf("expected 'template id is required', got %v", err)
	}
	if _, err := programTemplateIDArg([]string{"abc"}); err == nil || !strings.Contains(err.Error(), `invalid template id "abc"`) {
		t.Errorf("expected invalid template id, got %v", err)
	}
	if id, err := programTemplateIDArg([]string{"12"}); err != nil || id != 12 {
		t.Errorf("programTemplateIDArg(12) = %d, %v", id, err)
	}
}

// TestTemplateRequestFromFlags covers the shared config-file loader branches.
func TestTemplateRequestFromFlags(t *testing.T) {
	// Missing config file -> read error.
	if _, err := templateRequestFromFlags("n", "d", filepath.Join(t.TempDir(), "nope.json")); err == nil || !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("expected read-config error, got %v", err)
	}

	// Invalid JSON -> parse error.
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := templateRequestFromFlags("n", "d", bad); err == nil || !strings.Contains(err.Error(), "failed to parse config JSON") {
		t.Errorf("expected parse-config error, got %v", err)
	}

	// Empty config path -> flags only, no file access.
	req, err := templateRequestFromFlags("n", "d", "")
	if err != nil {
		t.Fatal(err)
	}
	if req.Name != "n" || req.Description != "d" || req.Config != nil {
		t.Errorf("expected flags-only request, got %+v", req)
	}
}
