package pipeline

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// The program `pipeline` CRUD commands are project-scoped and do NOT perform the
// billing check: each op is exactly 1 request (or 0 when validation / confirmation
// aborts before the client call).

// TestProgramPipelineCreate asserts `program pipeline create` posts the PipelineRequest
// and prints the confirmation line carrying the derived id/name.
func TestProgramPipelineCreate(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "build-all"}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "create", "-E", "2", "-P", "423",
		"--name", "build-all")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Created program pipeline 706 (build-all)\n") {
		t.Errorf("expected created line, got:\n%s", out)
	}
	if !strings.Contains(gotBody, `"name":"build-all"`) {
		t.Errorf("expected name in body, got %s", gotBody)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines") {
		t.Errorf("expected pipelines path, got %s", gotPaths[0])
	}
}

// TestProgramPipelineCreateRequiresName asserts the --name gate fires before any
// request.
func TestProgramPipelineCreateRequiresName(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "create", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected --name requirement, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before validation, got %d", reqCount)
	}
}

// TestProgramPipelineCreateBodyFile asserts create --body reads the JSON file and
// that --name wins over the file's name field.
func TestProgramPipelineCreateBodyFile(t *testing.T) {
	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "pipeline.json")
	if err := os.WriteFile(bodyFile, []byte(`{"name":"from-file","parameters":[{"key":"GITEE_BRANCH"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "cli-name"}))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "create", "-E", "2", "-P", "423",
		"--body", bodyFile, "--name", "cli-name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"name":"cli-name"`) {
		t.Errorf("expected --name to override body, got %s", gotBody)
	}
	if !strings.Contains(gotBody, `"parameters":[{"key":"GITEE_BRANCH"}]`) {
		t.Errorf("expected parameters kept from body, got %s", gotBody)
	}
}

// TestProgramPipelineCreateConfigFile asserts create --config reads the JSON
// file and that --name overrides the file's name field (same path as --body).
func TestProgramPipelineCreateConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "pipeline.json")
	// A full stage with a job so the config survives the round-trip (empty
	// slices are dropped by omitempty).
	if err := os.WriteFile(cfgFile, []byte(`{"name":"from-config","stages":[{"name":"build","jobs":[[{"name":"compile","identifier":"compile","type":"JENKINS_JOB","data":{"wait":true}}]]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "cli-name"}))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "create", "-E", "2", "-P", "423",
		"--config", cfgFile, "--name", "cli-name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"name":"cli-name"`) {
		t.Errorf("expected --name to override config, got %s", gotBody)
	}
	if !strings.Contains(gotBody, `"JENKINS_JOB"`) {
		t.Errorf("expected stages kept from config, got %s", gotBody)
	}
}

// TestProgramPipelineCreateBodyStdin asserts create --body - reads the JSON from
// stdin.
func TestProgramPipelineCreateBodyStdin(t *testing.T) {
	var mu sync.Mutex
	var gotBody string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotBody = readBody(t, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "from-stdin"}))
	}, nil)
	f.IOStreams.In = io.NopCloser(bytes.NewReader([]byte(`{"name":"from-stdin","stages":[{"name":"s","jobs":[[{"name":"j","identifier":"j","type":"JENKINS_JOB","data":{}}]]}]}`)))

	_, _, err := runPipelineCmd(t, f, "program", "create", "-E", "2", "-P", "423", "--body", "-")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"name":"from-stdin"`) {
		t.Errorf("expected body from stdin, got %s", gotBody)
	}
	if !strings.Contains(gotBody, `"JENKINS_JOB"`) {
		t.Errorf("expected stages carried from stdin, got %s", gotBody)
	}
}

// TestProgramPipelineCreateEmptyRequestWarns asserts a bare --name request
// (no stages) still submits (advisory on stderr) and is not rejected client-side.
func TestProgramPipelineCreateEmptyRequestWarns(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "test"}))
	}, nil)

	_, errOut, err := runPipelineCmd(t, f, "program", "create", "-E", "2", "-P", "423", "--name", "test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut, "at least one stage with a job") {
		t.Errorf("expected advisory hint on stderr, got %q", errOut)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 1 {
		t.Fatalf("expected 1 request (advisory only), got %d", reqCount)
	}
}

// TestProgramPipelineUpdateNothingToUpdate asserts the empty-edit guard fires
// before the client call (no flags -> no request).
func TestProgramPipelineUpdateNothingToUpdate(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "edit", "706", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "nothing to edit: pass --name or --body") {
		t.Errorf("expected nothing-to-edit error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request for empty edit, got %d", reqCount)
	}
}

// TestProgramPipelineUpdate asserts `pipeline edit --name` PUTs to pipelines/{id}
// and prints the confirmation line.
func TestProgramPipelineUpdate(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "new-name"}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "edit", "706", "-E", "2", "-P", "423", "--name", "new-name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Edited program pipeline 706 (new-name)\n") {
		t.Errorf("expected edited line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodPut || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/706") {
		t.Errorf("expected PUT pipelines/706, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramPipelineClone asserts `pipeline clone` POSTs to pipelines/{id}/clone
// and carries both source and derived ids in the confirmation line.
func TestProgramPipelineClone(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.707", Name: "build-all-copy"}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "clone", "706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Cloned program pipeline 707 from 706\n") {
		t.Errorf("expected cloned line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/706/clone") {
		t.Errorf("expected POST clone path, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramPipelineDeleteRequiresYes asserts the confirmation gate fires before
// the client call in a non-interactive context (0 requests).
func TestProgramPipelineDeleteRequiresYes(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "delete", "706", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--yes is required in non-interactive mode") {
		t.Errorf("expected confirmation error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before confirmation, got %d", reqCount)
	}
}

// TestProgramPipelineDeleteYes asserts the destructive delete with --yes issues the
// DELETE and prints the confirmation line.
func TestProgramPipelineDeleteYes(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotMethod string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, true)) // DeleteProgramPipeline decodes ResultVO[bool]
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "delete", "706", "-E", "2", "-P", "423", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Deleted program pipeline 706\n") {
		t.Errorf("expected deleted line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotMethod != http.MethodDelete || !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/706") {
		t.Errorf("expected DELETE pipelines/706, method=%s path=%s", gotMethod, gotPaths[0])
	}
}

// TestProgramPipelineHistory asserts a non-empty history renders the version rows
// from the multi-source history endpoint.
func TestProgramPipelineHistory(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineHistoryVO]{
			Current:  1,
			PageSize: 20,
			Total:    1,
			Data: []giteego.PipelineHistoryVO{{
				ID:         9,
				Version:    3,
				Hits:       42,
				Creator:    "alice",
				Online:     true,
				CreateTime: mustTime(t, "2026-08-17 10:00:00"),
			}},
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "history", "706", "-E", "2", "-P", "423")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Total: 1") {
		t.Errorf("expected total, got:\n%s", out)
	}
	for _, want := range []string{
		"VERSION  HITS  CREATOR      CREATED             ONLINE",
		"3",
		"alice",
		"2026-08-17 10:00:00",
		"yes",
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
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/706/history") {
		t.Errorf("expected history path, got %s", gotPaths[0])
	}
}

// TestProgramPipelineHistoryApplyRequiresHistory asserts the --history gate fires
// before any request.
func TestProgramPipelineHistoryApplyRequiresHistory(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "history-apply", "706", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "--history <history-id> is required") {
		t.Errorf("expected --history requirement, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before validation, got %d", reqCount)
	}
}

// TestProgramPipelineHistoryApplyRequiresYes asserts the confirmation gate fires
// for the destructive apply without --yes (0 requests).
func TestProgramPipelineHistoryApplyRequiresYes(t *testing.T) {
	var mu sync.Mutex
	var reqCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "history-apply", "706", "-E", "2", "-P", "423", "--history", "3")
	if err == nil || !strings.Contains(err.Error(), "--yes is required in non-interactive mode") {
		t.Errorf("expected confirmation error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if reqCount != 0 {
		t.Fatalf("expected no request before confirmation, got %d", reqCount)
	}
}

// TestProgramPipelineHistoryApplyYes asserts the apply flow first verifies the
// history record belongs to the pipeline, then POSTs history/{id}/apply.
func TestProgramPipelineHistoryApplyYes(t *testing.T) {
	var mu sync.Mutex
	var gotRequests []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotRequests = append(gotRequests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pipelines/706/history") {
			_, _ = w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineHistoryVO]{
				Current: 1, PageSize: 100, Total: 1,
				Data: []giteego.PipelineHistoryVO{{ID: 3, Version: 2}},
			}))
			return
		}
		_, _ = w.Write(jsonBody(t, &giteego.PipelineVO{Identifier: "pipeline.ops.pipeline.706", Name: "build-all"}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "program", "history-apply", "706", "-E", "2", "-P", "423", "--history", "3", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Applied history 3 to program pipeline 706 (build-all)\n") {
		t.Errorf("expected apply line, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotRequests) != 2 {
		t.Fatalf("expected ownership check + apply, got %v", gotRequests)
	}
	if !strings.Contains(gotRequests[0], "GET /2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/706/history") {
		t.Errorf("expected history ownership check first, got %s", gotRequests[0])
	}
	if !strings.Contains(gotRequests[1], "POST /2/423/gitee-go/ipipe/rest/v5/multi-source/pipelines/history/3/apply") {
		t.Errorf("expected POST history/3/apply, got %s", gotRequests[1])
	}
}

// TestProgramPipelineHistoryApplyRejectsForeignHistory asserts that a history
// record absent from the pipeline's (fully scanned) history listing aborts
// before the apply request.
func TestProgramPipelineHistoryApplyRejectsForeignHistory(t *testing.T) {
	var mu sync.Mutex
	var gotRequests []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotRequests = append(gotRequests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineHistoryVO]{
			Current: 1, PageSize: 100, Total: 1,
			Data: []giteego.PipelineHistoryVO{{ID: 9, Version: 1}},
		}))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "history-apply", "706", "-E", "2", "-P", "423", "--history", "3", "--yes")
	if err == nil || !strings.Contains(err.Error(), "does not belong to program pipeline 706") {
		t.Errorf("expected ownership rejection, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotRequests) != 1 {
		t.Fatalf("expected only the history check, no apply, got %v", gotRequests)
	}
}

// TestProgramPipelineHistoryApplyUnverifiedScanProceedsWithWarning asserts that
// when the history listing is too deep to rule out ownership, the apply still
// proceeds after a warning on stderr.
func TestProgramPipelineHistoryApplyUnverifiedScanProceedsWithWarning(t *testing.T) {
	var mu sync.Mutex
	var applyCount int
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			mu.Lock()
			applyCount++
			mu.Unlock()
			_, _ = w.Write(jsonBody(t, &giteego.PipelineVO{Name: "build-all"}))
			return
		}
		data := make([]giteego.PipelineHistoryVO, 100)
		for i := range data {
			data[i] = giteego.PipelineHistoryVO{ID: int64(10000 + i)}
		}
		_, _ = w.Write(jsonBody(t, giteego.PageVO[giteego.PipelineHistoryVO]{
			Current: 1, PageSize: 100, Total: 2050, Data: data,
		}))
	}, nil)

	_, errOut, err := runPipelineCmd(t, f, "program", "history-apply", "706", "-E", "2", "-P", "423", "--history", "1500", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut, "could not confirm history 1500") {
		t.Errorf("expected warning on stderr, got:\n%s", errOut)
	}
	mu.Lock()
	defer mu.Unlock()
	if applyCount != 1 {
		t.Fatalf("expected apply to proceed, got %d", applyCount)
	}
}

// TestProgramPipelineMissingArg asserts cobra's arity validation fires first.
func TestProgramPipelineMissingArg(t *testing.T) {
	_, _, err := runPipelineCmd(t, newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}, nil), "program", "view", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra arity error, got %v", err)
	}
}

// The helpers below (schemeValueString, typedValue, schemeSelectOptions,
// seedMultiSelection, schemeValidate, isGiteeGoQuota, friendlyAPIError) back
// the interactive wizard's scheme-driven data form. They are pure functions
// and are tested without a terminal.

func TestSchemeValueString(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{nil, ""},
		{"7", "7"},
		{true, "true"},
		{false, "false"},
		{float64(8), "8"},
		{float64(3.5), "3.5"},
		{float64(-1.25), "-1.25"},
		{[]interface{}{"a"}, "[a]"},
	}
	for _, c := range cases {
		if got := schemeValueString(c.in); got != c.want {
			t.Errorf("schemeValueString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTypedValue(t *testing.T) {
	if v, ok := typedValue("Number", "42").(int64); !ok || v != 42 {
		t.Errorf("expected int64 42, got %#v", typedValue("Number", "42"))
	}
	if v, ok := typedValue("Number", "3.14").(float64); !ok || v != 3.14 {
		t.Errorf("expected float64 3.14, got %#v", typedValue("Number", "3.14"))
	}
	if v := typedValue("Number", ""); v != "" {
		t.Errorf("empty number should stay empty string, got %#v", v)
	}
	if v := typedValue("Input", "abc"); v != "abc" {
		t.Errorf("plain input stays string, got %#v", v)
	}
}

func TestSchemeSelectOptions(t *testing.T) {
	c := giteego.PluginComponentScheme{
		Identifier:   "jdkVersion",
		Type:         "Select",
		DefaultValue: "8",
		Options: []giteego.ComponentOptionVO{
			{Label: "JDK 7", Value: "7"},
			{Label: "JDK 8", Value: "8"},
			{Key: "k9", Label: "JDK 9", Value: "9"},
		},
	}
	opts, raw, defKey := schemeSelectOptions(c)
	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	if defKey != "8" {
		t.Errorf("expected default key 8, got %q", defKey)
	}
	if raw["9"] != "9" || raw["8"] != "8" {
		t.Errorf("expected raw values preserved, got %v", raw)
	}
	if opts[0].Key != "JDK 7" {
		t.Errorf("expected label as key text, got %q", opts[0].Key)
	}
}

func TestSeedMultiSelection(t *testing.T) {
	hasAll := func(got []string, want ...string) bool {
		if len(got) != len(want) {
			return false
		}
		for _, w := range want {
			found := false
			for _, g := range got {
				if g == w {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	c := giteego.PluginComponentScheme{DefaultValue: []interface{}{"a", "c"}}
	got := seedMultiSelection(c)
	if !hasAll(got, "a", "c") {
		t.Errorf("array default: expected {a c}, got %v", got)
	}

	c = giteego.PluginComponentScheme{DefaultValue: map[string]interface{}{
		"a": true, "b": false, "c": true,
	}}
	got = seedMultiSelection(c)
	if !hasAll(got, "a", "c") {
		t.Errorf("object default: expected {a c}, got %v", got)
	}

	c = giteego.PluginComponentScheme{DefaultValue: "b"}
	got = seedMultiSelection(c)
	if !hasAll(got, "b") {
		t.Errorf("single default: expected {b}, got %v", got)
	}
}

func TestSchemeValidate(t *testing.T) {
	c := giteego.PluginComponentScheme{
		Rules: []giteego.PluginValidateRuleVO{
			{Required: true, ErrorMsg: "required-field"},
			{Regex: `^[a-z0-9_-]+$`, ErrorMsg: "bad-format"},
		},
	}
	v := schemeValidate(c, "REQUIRED")
	if err := v(""); err == nil || err.Error() != "required-field" {
		t.Errorf("expected required error, got %v", err)
	}
	if err := v("UPPER!"); err == nil || err.Error() != "bad-format" {
		t.Errorf("expected regex error, got %v", err)
	}
	if err := v("ok_1"); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestFriendlyAPIError(t *testing.T) {
	quota := &giteego.APIError{StatusCode: 429, Message: "quota exceeded", Code: "insufficient_quota"}
	err := friendlyAPIError(fmt.Errorf("failed to list plugins: %w", quota))
	if !strings.Contains(err.Error(), "insufficient_quota") {
		t.Errorf("expected quota label in %q", err.Error())
	}
	if !strings.Contains(err.Error(), "quota") {
		t.Errorf("expected translated hint in %q", err.Error())
	}
	if !isGiteeGoQuota(err) {
		t.Errorf("expected isGiteeGoQuota true for %v", err)
	}

	perm := &giteego.APIError{StatusCode: 403, Message: "forbidden"}
	if isGiteeGoQuota(fmt.Errorf("wrap: %w", perm)) {
		t.Errorf("403 should not be quota")
	}
	friendly := friendlyAPIError(fmt.Errorf("wrap: %w", perm))
	if !strings.Contains(friendly.Error(), "forbidden") {
		t.Errorf("permission error must keep its message, got %q", friendly.Error())
	}
	if !strings.Contains(friendly.Error(), "\n") {
		t.Errorf("permission error should carry an appended hint, got %q", friendly.Error())
	}

	if friendlyAPIError(errors.New("plain")) == nil {
		t.Errorf("non-API errors pass through")
	}
}

// TestProgramPipelineCreateQuotaHint asserts a gitee-go insufficient_quota
// (HTTP 429) response surfaces a translated, actionable hint instead of a raw
// JSON dump.
func TestProgramPipelineCreateQuotaHint(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"You exceeded your current quota","type":"insufficient_quota","code":"insufficient_quota"}`))
	}, nil)

	_, errOut, err := runPipelineCmd(t, f, "program", "create", "-E", "2", "-P", "423",
		"--config", "testdata/quota-pipeline.json")
	if err == nil {
		t.Fatal("expected quota error")
	}
	if !strings.Contains(err.Error(), "giteego: HTTP 429") {
		t.Errorf("expected HTTP 429 in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "quota") {
		t.Errorf("expected quota detail in error, got: %v", err)
	}
	if !strings.Contains(errOut, "quota") {
		t.Errorf("expected friendly quota hint on stderr, got: %q", errOut)
	}
}

func TestComponentLabelStripsTemplates(t *testing.T) {
	// i18n template name -> yaml field fallback.
	c := giteego.PluginComponentScheme{
		Name:      "#{{build.maven.jdkVersion.displayName}}",
		Convertor: &giteego.PluginFieldConvertorVO{YAMLField: "jdkVersion"},
	}
	if got := componentLabel(c); got != "jdkVersion" {
		t.Errorf("template label should fall back to yaml field, got %q", got)
	}
	// Plain name is kept.
	c2 := giteego.PluginComponentScheme{Name: "压缩类型", Identifier: "type"}
	if got := componentLabel(c2); got != "压缩类型" {
		t.Errorf("plain label kept, got %q", got)
	}
	// Identifier fallback.
	c3 := giteego.PluginComponentScheme{Identifier: "COMMAND"}
	if got := componentLabel(c3); got != "COMMAND" {
		t.Errorf("identifier fallback, got %q", got)
	}
}

func TestUsableDefaultSkipsTemplates(t *testing.T) {
	if d := usableDefault("#{{build.maven.commands.default}}"); d != nil {
		t.Errorf("i18n template default should be dropped, got %v", d)
	}
	if d := usableDefault("{{certificate}}"); d != "{{certificate}}" {
		t.Errorf("single-brace user ref default kept, got %v", d)
	}
	if d := usableDefault("mvn clean install"); d != "mvn clean install" {
		t.Errorf("plain default kept, got %v", d)
	}
	if !isTemplateString("#{{x.y}}") {
		t.Errorf("expected template detected")
	}
	if isTemplateString("{{x.y}}") {
		t.Errorf("single-brace ref must not be treated as i18n template")
	}
}

func TestSchemePlaceholderStripsTemplates(t *testing.T) {
	c := giteego.PluginComponentScheme{Placeholder: "#{{plugin.artifacts.placeholder}}"}
	if got := schemePlaceholder(c); got == "" || got == "#{{plugin.artifacts.placeholder}}" {
		t.Errorf("template placeholder should be replaced, got %q", got)
	}
	c2 := giteego.PluginComponentScheme{Placeholder: "owner/repo"}
	if got := schemePlaceholder(c2); got != "owner/repo" {
		t.Errorf("plain placeholder kept, got %q", got)
	}
}

func TestAssembleJobDataParameters(t *testing.T) {
	// Mirrors gitee-go's Converter.toFormGeneric: fields with convertor.parameters
	// (true or unset) go into data.parameters as {key,value} entries in scheme
	// order; explicit parameters:false goes to data.<identifier>. Password
	// components additionally carry "type":"PASSWORD" on their entry.
	fields := []collectedComponent{
		{comp: giteego.PluginComponentScheme{Identifier: "PLUGIN_JAVA_VERSION", Type: "RemoteSelect", Convertor: &giteego.PluginFieldConvertorVO{Parameters: true, YAMLField: "jdkVersion"}}, val: "8"},
		{comp: giteego.PluginComponentScheme{Identifier: "PLUGIN_TOKEN", Type: "Password", Convertor: &giteego.PluginFieldConvertorVO{Parameters: true, YAMLField: "token"}}, val: "s3cr3t"},
		{comp: giteego.PluginComponentScheme{Identifier: "COMMAND", Type: "Command"}, val: "mvn clean"},
		{comp: giteego.PluginComponentScheme{Identifier: "AGENT", Type: "Input", Convertor: &giteego.PluginFieldConvertorVO{Parameters: false, YAMLField: "agent"}}, val: "host-1"},
	}
	data, err := assembleJobData(fields)
	if err != nil {
		t.Fatal(err)
	}
	params, ok := data["parameters"].([]jobParameterVO)
	if !ok || len(params) != 3 {
		t.Fatalf("expected 3 parameters entries, got %#v", data["parameters"])
	}
	if params[0].Key != "PLUGIN_JAVA_VERSION" || params[0].Value != "8" || params[0].Type != "" {
		t.Errorf("entry 0 = %+v (non-password must not carry type)", params[0])
	}
	if params[1].Key != "PLUGIN_TOKEN" || params[1].Value != "s3cr3t" || params[1].Type != "PASSWORD" {
		t.Errorf("entry 1 = %+v, want key PLUGIN_TOKEN type PASSWORD", params[1])
	}
	if params[2].Key != "COMMAND" || params[2].Value != "mvn clean" || params[2].Type != "" {
		t.Errorf("entry 2 = %+v", params[2])
	}
	if data["AGENT"] != "host-1" {
		t.Errorf("parameters:false should sit at data.AGENT, got %#v", data["AGENT"])
	}
}

func TestJobParamValueConversions(t *testing.T) {
	// Switch → "true"/"false" string (toString JSON.stringify boolean).
	if v := jobParamValue(giteego.PluginComponentScheme{Type: "Switch"}, true); v != "true" {
		t.Errorf("switch true = %v", v)
	}
	if v := jobParamValue(giteego.PluginComponentScheme{Type: "Switch"}, false); v != "false" {
		t.Errorf("switch false = %v", v)
	}
	// Compose array → JSON string.
	artifacts := []interface{}{
		map[string]interface{}{"name": "BUILD_ARTIFACT", "path": []interface{}{"./target"}, "type": ".tar.gz"},
	}
	got := jobParamValue(giteego.PluginComponentScheme{Type: "Compose", Array: true}, artifacts)
	expected := `[{"name":"BUILD_ARTIFACT","path":["./target"],"type":".tar.gz"}]`
	if got != expected {
		t.Errorf("compose array stringify = %s", got)
	}
	// Multi-select array → JSON string.
	multi := jobParamValue(giteego.PluginComponentScheme{Type: "Select", Multiple: true}, []interface{}{"a", "b"})
	if multi != `["a","b"]` {
		t.Errorf("multiple stringify = %v", multi)
	}
	// Single Input → trimmed.
	if v := jobParamValue(giteego.PluginComponentScheme{Type: "Input"}, "  x  "); v != "x" {
		t.Errorf("input trim = %v", v)
	}
	// Scalars pass through; nil → empty string.
	if v := jobParamValue(giteego.PluginComponentScheme{Type: "Number"}, float64(8)); v != float64(8) {
		t.Errorf("number passthrough = %v", v)
	}
	if v := jobParamValue(giteego.PluginComponentScheme{Type: "Input"}, nil); v != "" {
		t.Errorf("nil = %v", v)
	}
}

func TestResolveRefValue(t *testing.T) {
	c := giteego.PluginComponentScheme{DefaultValue: "{{CERTIFICATION.${certificate}.server}}"}
	got := resolveRefValue(c, map[string]interface{}{"certificate": "cert-1"})
	if got != "{{CERTIFICATION.cert-1.server}}" {
		t.Errorf("resolved = %q", got)
	}
	// Unresolvable placeholders stay as-is; non-string defaults resolve to "".
	got = resolveRefValue(c, map[string]interface{}{})
	if got != "{{CERTIFICATION.${certificate}.server}}" {
		t.Errorf("unresolved should keep template, got %q", got)
	}
	if got := resolveRefValue(giteego.PluginComponentScheme{DefaultValue: float64(3)}, nil); got != "" {
		t.Errorf("non-string default = %q", got)
	}
}

func TestAssembleJobDataDuplicateIdentifier(t *testing.T) {
	// image-build style: same identifier as RefValue then Input — last wins
	// (frontend step map semantics).
	fields := []collectedComponent{
		{comp: giteego.PluginComponentScheme{Identifier: "DOCKER_USERNAME", Type: "RefValue", Convertor: &giteego.PluginFieldConvertorVO{Parameters: true}}, val: "{{params.user}}"},
		{comp: giteego.PluginComponentScheme{Identifier: "DOCKER_USERNAME", Type: "Input", Convertor: &giteego.PluginFieldConvertorVO{Parameters: true}}, val: "admin"},
	}
	data, err := assembleJobData(fields)
	if err != nil {
		t.Fatal(err)
	}
	params := data["parameters"].([]jobParameterVO)
	if len(params) != 1 || params[0].Value != "admin" {
		t.Fatalf("expected single last-wins entry, got %#v", params)
	}
}

// TestAssembleJobDataPasswordLastWins verifies the replacement branch of a
// duplicated identifier keeps the password type of the winning (last) variant.
func TestAssembleJobDataPasswordLastWins(t *testing.T) {
	fields := []collectedComponent{
		{comp: giteego.PluginComponentScheme{Identifier: "PLUGIN_TOKEN", Type: "Input", Convertor: &giteego.PluginFieldConvertorVO{Parameters: true}}, val: "visible"},
		{comp: giteego.PluginComponentScheme{Identifier: "PLUGIN_TOKEN", Type: "Password", Convertor: &giteego.PluginFieldConvertorVO{Parameters: true}}, val: "s3cr3t"},
	}
	data, err := assembleJobData(fields)
	if err != nil {
		t.Fatal(err)
	}
	params := data["parameters"].([]jobParameterVO)
	if len(params) != 1 || params[0].Key != "PLUGIN_TOKEN" ||
		params[0].Value != "s3cr3t" || params[0].Type != "PASSWORD" {
		t.Fatalf("expected last-wins password entry with type PASSWORD, got %#v", params)
	}
}
