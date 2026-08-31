package pipeline

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// The program `param` commands are project-scoped and do NOT perform the
// billing check, so each op is exactly 1 request (or 0 when validation /
// confirmation aborts before the client call).

func programParam(id int64, name string) giteego.ParamVO {
	return giteego.ParamVO{ID: id, Name: name}
}

// TestProgramParamView asserts the view layout (ID/Name/Description/params/
// Labels) and the multi-source params path with pathBase 2/423.
func TestProgramParamView(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.ParamVO{
			ID:          12,
			Name:        "build-params",
			Description: "shared build params",
			Parameters: []giteego.ParameterStruct{{
				Key: "GITEE_BRANCH", DefaultValue: "master",
			}},
			Labels: []giteego.LabelVO{{Name: "release"}, {Name: "nightly"}},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "program", "param", "view", "12", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if gotPathBase != "2/423" {
		t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
	}
	for _, want := range []string{
		"ID:          12\n",
		"Name:        build-params\n",
		"Description: shared build params\n",
		i18n.T("pipeline.params_pipeline") + ":\n",
		"    GITEE_BRANCH = master\n",
		"Labels\n",
		"  release\n",
		"  nightly\n",
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
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/params/12") {
		t.Errorf("expected program param path, got %q", gotPaths[0])
	}
}

// TestProgramParamViewNil asserts a nil param surfaces the not-found error.
func TestProgramParamViewNil(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "param", "view", "12", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "program param 12 not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestProgramParamCreate asserts `param create` posts the request and prints
// the confirmation line carrying the created id/name.
func TestProgramParamCreate(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, programParam(13, "build-params")))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "param", "create", "-E", "2", "-P", "423",
		"--name", "build-params", "--description", "shared build params")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Created program param 13 (build-params)\n") {
		t.Errorf("expected created line, got:\n%s", out)
	}
	if !strings.Contains(gotBody, `"name":"build-params"`) || !strings.Contains(gotBody, `"description":"shared build params"`) {
		t.Errorf("expected name+description in body, got %s", gotBody)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/params") {
		t.Errorf("expected params path, got %s", gotPaths[0])
	}
}

// TestProgramParamCreateRequiresName asserts the --name gate fires before any
// request.
func TestProgramParamCreateRequiresName(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "param", "create", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected --name requirement, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before validation, got %d", reqCount)
	}
}

// TestProgramParamUpdateNothingToUpdate asserts the empty-edit guard fires
// before the client call (no flags -> no request).
func TestProgramParamUpdateNothingToUpdate(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "param", "edit", "12", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "nothing to edit: pass --name, --description or --body") {
		t.Errorf("expected nothing-to-edit error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request for empty edit, got %d", reqCount)
	}
}

// TestProgramParamUpdate asserts `param edit --name` PUTs to params/{id} and
// prints the confirmation line.
func TestProgramParamUpdate(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, programParam(12, "new-name")))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "param", "edit", "12", "-E", "2", "-P", "423", "--name", "new-name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Edited program param 12 (new-name)\n") {
		t.Errorf("expected edited line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodPut || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/params/12") {
		t.Errorf("expected PUT params/12, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramParamClone asserts `param clone` PUTs to params/{id}/clone and
// carries both ids in the confirmation line.
func TestProgramParamClone(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, programParam(14, "build-params-copy")))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "param", "clone", "12", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Cloned program param 14 from 12\n") {
		t.Errorf("expected cloned line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/params/12/clone") {
		t.Errorf("expected clone path, got %s", gotPaths[0])
	}
}

// TestProgramParamBodyFile asserts create --body reads the JSON file and that
// --name wins over the file's name field.
func TestProgramParamBodyFile(t *testing.T) {
	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "params.json")
	if err := os.WriteFile(bodyFile, []byte(`{"name":"from-file","description":"from body"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, programParam(15, "from-file")))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "param", "create", "-E", "2", "-P", "423",
		"--body", bodyFile, "--name", "cli-name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"name":"cli-name"`) {
		t.Errorf("expected --name to override body, got %s", gotBody)
	}
	if !strings.Contains(gotBody, `"description":"from body"`) {
		t.Errorf("expected description kept from body, got %s", gotBody)
	}
}

// TestProgramParamMissingArg asserts cobra's arity validation fires first.
func TestProgramParamMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "program", "param", "view", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}

// TestProgramParamIDArg covers the programParamIDArg pure parser branches.
func TestProgramParamIDArg(t *testing.T) {
	if _, err := programParamIDArg(nil); err == nil || !strings.Contains(err.Error(), "param id is required") {
		t.Errorf("expected 'param id is required', got %v", err)
	}
	if _, err := programParamIDArg([]string{"abc"}); err == nil || !strings.Contains(err.Error(), `invalid param id "abc"`) {
		t.Errorf("expected invalid param id, got %v", err)
	}
	if id, err := programParamIDArg([]string{"12"}); err != nil || id != 12 {
		t.Errorf("programParamIDArg(12) = %d, %v", id, err)
	}
}

// TestParamRequestFromFlags covers the shared body-file loader branches.
func TestParamRequestFromFlags(t *testing.T) {
	// Missing body file -> read error.
	if _, err := paramRequestFromFlags("", "", filepath.Join(t.TempDir(), "nope.json")); err == nil || !strings.Contains(err.Error(), "failed to read body file") {
		t.Errorf("expected read-body error, got %v", err)
	}

	// Invalid JSON -> parse error.
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := paramRequestFromFlags("", "", bad); err == nil || !strings.Contains(err.Error(), "failed to parse body JSON") {
		t.Errorf("expected parse-body error, got %v", err)
	}

	// Empty body path -> flags only, no file access.
	req, err := paramRequestFromFlags("n", "d", "")
	if err != nil {
		t.Fatal(err)
	}
	if req.Name != "n" || req.Description != "d" {
		t.Errorf("expected flags-only request, got %+v", req)
	}
}
