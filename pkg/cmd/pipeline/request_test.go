package pipeline

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// TestPipelineRequestURLMode asserts an absolute URL fetches without -R and
// without a billing check, printing the trimmed body.
func TestPipelineRequestURLMode(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("  {\"line\":1}\n"))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "request", "https://premium-k8s.gitee.cn/2/426/gitee-go/logs?recordUuid=x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `{"line":1}`) {
		t.Errorf("expected trimmed body, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 1 {
		t.Fatalf("expected 1 request (no billing), got %v", gotPaths)
	}
	if gotPathBase != "" {
		t.Errorf("expected empty pathBase in URL mode, got %q", gotPathBase)
	}
}

// TestPipelineRequestURLModeHTTPError asserts a non-2xx absolute fetch surfaces
// the HTTP status.
func TestPipelineRequestURLModeHTTPError(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}, nil)

	_, _, err := runPipelineCmd(t, f, "request", "https://premium-k8s.gitee.cn/logs?recordUuid=x")
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("expected HTTP 403 error, got %v", err)
	}
}

// TestPipelineRequestComponentMode asserts the component branch resolves the
// scheme then calls the remote-select endpoint (billing + 2 calls).
func TestPipelineRequestComponentMode(t *testing.T) {
	var mu sync.Mutex
	var gotJobTypes []string
	var gotRemoteTypes []string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if strings.Contains(r.URL.Path, "/plugins/scheme") {
			mu.Lock()
			gotJobTypes = append(gotJobTypes, r.URL.Query().Get("jobType"))
			mu.Unlock()
			_, _ = w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
				"gcc-build@v1.0.0": {
					Type: giteego.PluginSchemeTypeVO{JSON: "gcc-build@v1.0.0", YAML: "build@gcc"},
					Config: []giteego.PluginComponentScheme{
						{
							Identifier: "PLUGIN_GCC_VERSION",
							Type:       "RemoteSelect",
							URL:        &giteego.RemoteSelectURLVO{GitOps: "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version?type=gcc"},
							Params:     map[string]string{"type": "gcc"},
						},
					},
				},
			}))
			return
		}
		mu.Lock()
		gotRemoteTypes = append(gotRemoteTypes, r.URL.Query().Get("type"))
		mu.Unlock()
		_, _ = w.Write(jsonBody(t, giteego.RemoteSelectResponse{
			Total: 1,
			List:  []giteego.ComponentOptionVO{{Key: "9.4", Label: "GCC 9.4", Value: "9.4"}},
		}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "request", "PLUGIN_GCC_VERSION",
		"-R", "owner/repo", "--type", "gcc-build@v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GCC 9.4") {
		t.Errorf("expected option row, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotJobTypes) != 1 || gotJobTypes[0] != "gcc-build@v1.0.0" {
		t.Errorf("expected scheme jobType query, got %v", gotJobTypes)
	}
	if len(gotRemoteTypes) != 1 || gotRemoteTypes[0] != "gcc" {
		t.Errorf("expected remote type=gcc query, got %v", gotRemoteTypes)
	}
}

// TestPipelineRequestNoScheme asserts a scheme miss surfaces an error.
func TestPipelineRequestNoScheme(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		// data:null — GetPluginScheme resolves a nil scheme.
		_, _ = w.Write(jsonBody(t, nil))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "request", "PLUGIN_JAVA_VERSION",
		"-R", "owner/repo", "--type", "nope-build@v1.0.0")
	if err == nil || !strings.Contains(err.Error(), `no scheme for job type "nope-build@v1.0.0"`) {
		t.Errorf("expected no-scheme error, got %v", err)
	}
}

// TestPipelineRequestNoComponent asserts a missing component identifier fails.
func TestPipelineRequestNoComponent(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
			"gcc-build@v1.0.0": {Type: giteego.PluginSchemeTypeVO{JSON: "gcc-build@v1.0.0"}},
		}))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "request", "PLUGIN_NOPE",
		"-R", "owner/repo", "--type", "gcc-build@v1.0.0")
	if err == nil || !strings.Contains(err.Error(), `no RemoteSelect component "PLUGIN_NOPE" in plugin "gcc-build@v1.0.0"`) {
		t.Errorf("expected no-component error, got %v", err)
	}
}

// TestPipelineRequestNoGitOpsURL asserts a RemoteSelect component without a
// gitOps URL surfaces a clear error.
func TestPipelineRequestNoGitOpsURL(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
			"gcc-build@v1.0.0": {
				Type: giteego.PluginSchemeTypeVO{JSON: "gcc-build@v1.0.0"},
				Config: []giteego.PluginComponentScheme{
					{Identifier: "PLUGIN_GCC_VERSION", Type: "RemoteSelect"},
				},
			},
		}))
	}, nil)

	_, _, err := runPipelineCmd(t, f, "request", "PLUGIN_GCC_VERSION",
		"-R", "owner/repo", "--type", "gcc-build@v1.0.0")
	if err == nil || !strings.Contains(err.Error(), `component "PLUGIN_GCC_VERSION" has no gitOps remote-select URL`) {
		t.Errorf("expected no-gitOps error, got %v", err)
	}
}

// TestPipelineRequestNoOptions asserts the no-options message on an empty list.
func TestPipelineRequestNoOptions(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if strings.Contains(r.URL.Path, "/plugins/scheme") {
			_, _ = w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
				"gcc-build@v1.0.0": {
					Type: giteego.PluginSchemeTypeVO{JSON: "gcc-build@v1.0.0"},
					Config: []giteego.PluginComponentScheme{
						{
							Identifier: "PLUGIN_GCC_VERSION",
							Type:       "RemoteSelect",
							URL:        &giteego.RemoteSelectURLVO{GitOps: "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version"},
						},
					},
				},
			}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.RemoteSelectResponse{Total: 0, List: []giteego.ComponentOptionVO{}}))
	}, nil)

	out, _, err := runPipelineCmd(t, f, "request", "PLUGIN_GCC_VERSION",
		"-R", "owner/repo", "--type", "gcc-build@v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No options returned by the remote endpoint.") &&
		!strings.Contains(out, "远程端点没有返回任何选项") &&
		!strings.Contains(out, "远端端点未返回选项。") {
		t.Errorf("expected no-options message, got:\n%s", out)
	}
}

// TestRequestPureFuncs covers the request.go pure helpers.
func TestRequestPureFuncs(t *testing.T) {
	// isAbsoluteURL
	if !isAbsoluteURL("https://gitee.com/x") || !isAbsoluteURL("http://localhost:8080/logs") {
		t.Error("expected http/https absolute URLs to be recognized")
	}
	if isAbsoluteURL("PLUGIN_JAVA_VERSION") || isAbsoluteURL("/relative/path") || isAbsoluteURL("") {
		t.Error("expected non-URL strings to be rejected")
	}

	// findRemoteSelectComponent: top-level and nested children.
	config := []giteego.PluginComponentScheme{
		{Identifier: "PLUGIN_VERSION", Type: "RemoteSelect"},
		{
			Identifier: "PLUGIN_COMPOSE",
			Type:       "Compose",
			Children:   []giteego.PluginComponentScheme{{Identifier: "name", Type: "Input"}},
		},
	}
	if c := findRemoteSelectComponent(config, "PLUGIN_VERSION"); c == nil || c.Identifier != "PLUGIN_VERSION" {
		t.Errorf("expected top-level component found, got %+v", c)
	}
	if c := findRemoteSelectComponent(config, "name"); c == nil || c.Identifier != "name" {
		t.Errorf("expected nested child found, got %+v", c)
	}
	if c := findRemoteSelectComponent(config, "MISSING"); c != nil {
		t.Errorf("expected nil for missing component, got %+v", c)
	}

	// expandRef: ${key} substitution, malformed pairs skipped.
	if got := expandRef("${certificate}", []string{"certificate=uuid-123"}); got != "uuid-123" {
		t.Errorf("expandRef = %q, want uuid-123", got)
	}
	if got := expandRef("prefix-${a}-${b}", []string{"a=1", "b=2"}); got != "prefix-1-2" {
		t.Errorf("expandRef = %q, want prefix-1-2", got)
	}
	if got := expandRef("${a}", []string{"malformed"}); got != "${a}" {
		t.Errorf("expandRef(malformed) = %q, want ${a}", got)
	}

	// componentValue: nil Value -> Key; empty string -> Key; string -> itself.
	if got := componentValue(giteego.ComponentOptionVO{Key: "k"}); got != "k" {
		t.Errorf("componentValue(nil) = %q, want k", got)
	}
	if got := componentValue(giteego.ComponentOptionVO{Key: "k", Value: ""}); got != "k" {
		t.Errorf("componentValue(empty) = %q, want k", got)
	}
	if got := componentValue(giteego.ComponentOptionVO{Key: "k", Value: "v"}); got != "v" {
		t.Errorf("componentValue(str) = %q, want v", got)
	}
	if got := componentValue(giteego.ComponentOptionVO{Key: "k", Value: 42}); got != "42" {
		t.Errorf("componentValue(int) = %q, want 42", got)
	}
}

// TestPipelineRequestInvalidParam asserts an unparseable -p is rejected after
// the scheme resolves.
func TestPipelineRequestInvalidParam(t *testing.T) {
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if strings.Contains(r.URL.Path, "/plugins/scheme") {
			_, _ = w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
				"gcc-build@v1.0.0": {
					Type: giteego.PluginSchemeTypeVO{JSON: "gcc-build@v1.0.0"},
					Config: []giteego.PluginComponentScheme{
						{
							Identifier: "PLUGIN_GCC_VERSION",
							Type:       "RemoteSelect",
							URL:        &giteego.RemoteSelectURLVO{GitOps: "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version"},
						},
					},
				},
			}))
			return
		}
		t.Fatal("expected no remote-select call")
	}, nil)

	_, _, err := runPipelineCmd(t, f, "request", "PLUGIN_GCC_VERSION",
		"-R", "owner/repo", "--type", "gcc-build@v1.0.0", "-p", "broken")
	if err == nil || !strings.Contains(err.Error(), `invalid param "broken", expected key=value`) {
		t.Errorf("expected invalid param error, got %v", err)
	}
}
