package pipeline

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// The program `plugin` commands are project-scoped and do NOT perform the
// billing open-check: each op is exactly 1 request (list/scheme/example). The
// scheme/example routes carry the jobType query param through the /multi-source
// segment.

// TestProgramPluginScheme asserts `program plugin scheme` renders the scheme
// via printProjectSchemes (name / Type+ YAML line / description / doc / config
// component) with pathBase 2/423.
func TestProgramPluginScheme(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotQuery url.Values
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotQuery = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		// Real backend returns a map keyed by the json type.
		w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
			"maven-build@v1.0.0": {
				Type:        giteego.PluginSchemeTypeVO{JSON: "maven-build@v1.0.0", YAML: "build@maven"},
				Name:        "Maven 构建",
				Description: "使用 Apache Maven 云端构建 Java 项目",
				Doc:         "https://docs.example/maven",
				Config: []giteego.PluginComponentScheme{{
					Identifier:   "PLUGIN_JAVA_VERSION",
					Type:         "RemoteSelect",
					Name:         "#{{build.maven.jdkVersion.displayName}}",
					DefaultValue: "8",
					Rules:        []giteego.PluginValidateRuleVO{{Required: true}},
					URL:          &giteego.RemoteSelectURLVO{GitOps: "/rest/v5/external/remote-select/plugin-tools/version"},
				}},
			},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "program", "plugin", "scheme", "-E", "2", "-P", "423", "--type", "maven-build@v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if gotPathBase != "2/423" {
		t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
	}
	for _, want := range []string{
		"Maven 构建\n",
		"Type: maven-build@v1.0.0   YAML: build@maven\n",
		"使用 Apache Maven 云端构建 Java 项目\n",
		"Doc:  https://docs.example/maven\n",
		"build.maven.jdkVersion.displayName  [RemoteSelect]  (required)\n",
		"    default: 8\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
	// i18n placeholder must be stripped, Go map noise must not appear.
	if strings.Contains(out, "#{{build.maven.jdkVersion.displayName}}") {
		t.Errorf("expected i18n key stripped from component name:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/plugins/scheme") {
		t.Errorf("expected program scheme path, got %s", gotPaths[0])
	}
	if gotQuery.Get("jobType") != "maven-build@v1.0.0" {
		t.Errorf("expected jobType=maven-build@v1.0.0, got %q", gotQuery.Get("jobType"))
	}
}

// TestProgramPluginSchemeNoScheme asserts the empty-schemes guard surfaces the
// no-scheme error.
func TestProgramPluginSchemeNoScheme(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "plugin", "scheme", "-E", "2", "-P", "423", "--type", "UNKNOWN")
	if err == nil || !strings.Contains(err.Error(), `no scheme for job type "UNKNOWN"`) {
		t.Errorf("expected no-scheme error, got %v", err)
	}
}

// TestProgramPluginSchemeRequiresType asserts cobra's required-flag validation
// fires before any request.
func TestProgramPluginSchemeRequiresType(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "plugin", "scheme", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "type" not set`) {
		t.Errorf("expected required --type error, got %v", err)
	}
}

// TestProgramPluginExample asserts `program plugin example` writes the YAML
// string plus newline and passes the jobType query param.
func TestProgramPluginExample(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotQuery url.Values
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotQuery = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "- step: build@maven\n  name: build-maven-example"))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "program", "plugin", "example", "-E", "2", "-P", "423", "--type", "maven-build@v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if gotPathBase != "2/423" {
		t.Errorf("expected pathBase 2/423, got %q", gotPathBase)
	}
	if !strings.Contains(out, "step: build@maven\n") {
		t.Errorf("expected YAML in output, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "/2/423/gitee-go/ipipe/rest/v5/multi-source/plugins/example") {
		t.Errorf("expected program example path, got %s", gotPaths[0])
	}
	if gotQuery.Get("jobType") != "maven-build@v1.0.0" {
		t.Errorf("expected jobType=maven-build@v1.0.0, got %q", gotQuery.Get("jobType"))
	}
}

// TestProgramPluginExampleRequiresType asserts cobra's required-flag validation
// fires before any request.
func TestProgramPluginExampleRequiresType(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBody(t, "ok"))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "program", "plugin", "example", "-E", "2", "-P", "423")
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "type" not set`) {
		t.Errorf("expected required --type error, got %v", err)
	}
}
