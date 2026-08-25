package pipeline

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newTestFactory wires Factory with an in-memory go-api transport. The handler
// receives every go-api request (billing and pipeline endpoints share the
// client); billing requests return enabled=true by default. pathBaseOut, when
// non-nil, receives the repo path passed to GoAPIClient ("owner/repo").
func newTestFactory(t *testing.T, handler http.HandlerFunc, pathBaseOut *string) *cmdutil.Factory {
	t.Helper()
	// Pin the locale so tests are independent of the shell's LANG/LC_ALL
	// (assertions use English output).
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "en_US.UTF-8")
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    outBuf,
		ErrOut: errBuf,
	}
	f := &cmdutil.Factory{
		IOStreams: ios,
		Hostname:  "gitee.com",
		GoAPIClient: func(pathBase, service string) (*giteego.Client, error) {
			if pathBaseOut != nil {
				*pathBaseOut = pathBase
			}
			return giteego.NewClient("test-token",
				giteego.WithBaseURL("https://go-api.example/"+strings.Trim(pathBase, "/")+"/gitee-go"),
				giteego.WithServicePath(service),
				giteego.WithMaxRetries(0),
				giteego.WithHTTPClient(&http.Client{
					Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						rec := httptest.NewRecorder()
						handler(rec, r)
						return rec.Result(), nil
					}),
				}),
			), nil
		},
		GiteeClient: func() (*gitee.Client, error) {
			return gitee.NewClient("test-token", gitee.WithBaseURL("https://gitee.example/api/v5")), nil
		},
	}
	f.Context = t.Context()
	return f
}

func runPipelineCmd(t *testing.T, f *cmdutil.Factory, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewPipelineCmd(f)
	cmd.SetOut(f.IOStreams.Out)
	cmd.SetErr(f.IOStreams.ErrOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return f.IOStreams.Out.(*bytes.Buffer).String(), f.IOStreams.ErrOut.(*bytes.Buffer).String(), err
}

func jsonBody(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]interface{}{"code": 0, "msg": "", "data": v})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// readBody reads and trims a request body for assertions.
func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

func TestPipelineList(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, []giteego.PipelineYamlSummaryVO{
			{FileName: "ci.yml", Name: "CI"},
			{FileName: "release.yml", Name: "Release"},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "list", "-R", "owner/repo", "--ref", "master")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ci.yml") || !strings.Contains(out, "release.yml") {
		t.Errorf("unexpected output: %s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+list), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "billing/gitee-go-service/status") {
		t.Errorf("expected billing check first, got %s", gotPaths[0])
	}
	// The repo path must sit between the host and /gitee-go in the pipeline URL.
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines") {
		t.Errorf("expected pipelines list to include repo path, got %s", gotPaths[1])
	}
}

func TestPipelineRun(t *testing.T) {
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, map[string]interface{}{
			"id": 7, "buildNumber": 12, "status": "WAITTING", "fileName": "ci.yml", "ref": "master",
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "run", "-R", "owner/repo", "--ref", "master", "--file", "ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Build #12") || !strings.Contains(out, "WAITTING") {
		t.Errorf("unexpected output: %s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
}

func TestPipelinePluginList(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, []giteego.CategoryPluginVO{
			{
				Name: "工具",
				Plugins: []giteego.PluginVO{
					{Name: "Jenkins 任务", Type: "JENKINS_JOB", Description: "Run a Jenkins job"},
					{Name: "构建 Maven", Type: "BUILD_MAVEN", Description: "Build a Maven project"},
				},
			},
			{
				Name: "部署",
				Plugins: []giteego.PluginVO{
					{Name: "主机部署", Type: "GITEE_DEPLOY", Description: "Deploy to a host"},
				},
			},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "plugin", "list", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "工具") || !strings.Contains(out, "Jenkins 任务") || !strings.Contains(out, "主机部署") {
		t.Errorf("unexpected output: %s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+list), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[0], "billing/gitee-go-service/status") {
		t.Errorf("expected billing check first, got %s", gotPaths[0])
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/plugins") {
		t.Errorf("expected plugins list to include repo path, got %s", gotPaths[1])
	}
}

func TestPipelinePluginScheme(t *testing.T) {
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
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PluginSchemeVO{
			Type:        giteego.PluginSchemeTypeVO{JSON: "BUILD_MAVEN", YAML: "build@maven"},
			Name:        "Maven 构建",
			Description: "使用 Apache Maven 云端构建 Java 项目",
			Advance:     &giteego.PluginAdvanceSchemeVO{Notification: true, Timeout: true, Skip: true, Retry: true},
			Config: []giteego.PluginComponentScheme{
				{
					Identifier:   "PLUGIN_JAVA_VERSION",
					Type:         "RemoteSelect",
					Name:         "#{{build.maven.jdkVersion.displayName}}",
					DefaultValue: "8",
					Rules:        []giteego.PluginValidateRuleVO{{Required: true}},
					Convertor:    &giteego.PluginFieldConvertorVO{Parameters: true, YAMLField: "jdkVersion"},
					URL:          &giteego.RemoteSelectURLVO{GitOps: "/rest/v5/external/remote-select/plugin-tools/version"},
					Params:       map[string]string{"type": "open-jdk"},
				},
				{
					Identifier:   "PLUGIN_ARTIFACTS",
					Type:         "Compose",
					Name:         "#{{plugin.artifacts}}",
					DefaultValue: []map[string]interface{}{{"name": "BUILD_ARTIFACT", "path": []string{"./target"}}},
					Children:     []giteego.PluginComponentScheme{{Identifier: "name", Type: "Input", Name: "#{{plugin.artifacts.displayName}}"}},
				},
			},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "plugin", "scheme", "-R", "owner/repo", "--type", "BUILD_MAVEN")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Maven 构建") || !strings.Contains(out, "BUILD_MAVEN") {
		t.Errorf("unexpected output: %s", out)
	}
	// New layout: i18n keys stripped to the key, JSON default for composites,
	// required marker, section header, blank lines between components.
	if strings.Contains(out, "#{{build.maven.jdkVersion.displayName}}") {
		t.Errorf("expected i18n key stripped from component name: %s", out)
	}
	if strings.Contains(out, "[map[") {
		t.Errorf("expected Go map noise removed from default: %s", out)
	}
	if !strings.Contains(out, "build.maven.jdkVersion.displayName") {
		t.Errorf("expected readable i18n key in output: %s", out)
	}
	if !strings.Contains(out, "(required)") {
		t.Errorf("expected required marker: %s", out)
	}
	if !strings.Contains(out, "--------") {
		t.Errorf("expected section divider: %s", out)
	}
	if !strings.Contains(out, `"name":"BUILD_ARTIFACT"`) {
		t.Errorf("expected JSON default for compose: %s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+scheme), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/plugins/scheme") {
		t.Errorf("expected scheme path, got %s", gotPaths[1])
	}
	if gotQuery.Get("jobType") != "BUILD_MAVEN" {
		t.Errorf("expected jobType=BUILD_MAVEN, got %q", gotQuery.Get("jobType"))
	}
}

func TestPipelinePluginExample(t *testing.T) {
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
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, "- step: build@maven\n  name: build-maven-example"))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "plugin", "example", "-R", "owner/repo", "--type", "BUILD_MAVEN")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "step: build@maven") {
		t.Errorf("unexpected output: %s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+example), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/plugins/example") {
		t.Errorf("expected example path, got %s", gotPaths[1])
	}
	if gotQuery.Get("jobType") != "BUILD_MAVEN" {
		t.Errorf("expected jobType=BUILD_MAVEN, got %q", gotQuery.Get("jobType"))
	}
}

func TestPipelinePluginRequest(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotQueries []url.Values
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		gotQueries = append(gotQueries, r.URL.Query())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if strings.Contains(r.URL.Path, "/plugins/scheme") {
			// Real backend returns a map keyed by the json type.
			_, _ = w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
				"maven-build@v1.0.0": {
					Type:        giteego.PluginSchemeTypeVO{JSON: "maven-build@v1.0.0", YAML: "build@maven"},
					Name:        "Maven 构建",
					Description: "使用 Apache Maven 云端构建 Java 项目",
					Config: []giteego.PluginComponentScheme{
						{
							Identifier:   "PLUGIN_JAVA_VERSION",
							Type:         "RemoteSelect",
							DefaultValue: "8",
							// Realistic scheme value: full gateway route with embedded query.
							URL:    &giteego.RemoteSelectURLVO{GitOps: "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version?type=open-jdk"},
							Params: map[string]string{"type": "open-jdk"},
						},
					},
				},
			}))
			return
		}
		// The remote-select call: object form {total, list}.
		_, _ = w.Write(jsonBody(t, giteego.RemoteSelectResponse{
			Total: 2,
			List: []giteego.ComponentOptionVO{
				{Key: "8", Label: "OpenJDK 8", Value: "8"},
				{Key: "11", Label: "OpenJDK 11", Value: "11"},
			},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "request", "PLUGIN_JAVA_VERSION",
		"-R", "owner/repo", "--type", "maven-build@v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "OpenJDK 8") || !strings.Contains(out, "OpenJDK 11") {
		t.Errorf("unexpected output: %s", out)
	}
	if gotPathBase != "owner/repo" {
		t.Errorf("expected pathBase owner/repo, got %q", gotPathBase)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 3 {
		t.Fatalf("expected 3 requests (billing+scheme+remote), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/plugins/scheme") {
		t.Errorf("expected scheme path, got %s", gotPaths[1])
	}
	if !strings.Contains(gotPaths[2], "/owner/repo/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version") {
		t.Errorf("expected remote-select path, got %s", gotPaths[2])
	}
	if strings.Contains(gotPaths[2], "gitee-go/ipipe/gitee-go") || strings.Contains(gotPaths[2], "/ipipe/ipipe") {
		t.Errorf("expected no doubled gateway/service segment, got %s", gotPaths[2])
	}
	// The scheme's url.params {type: open-jdk} must reach the remote endpoint.
	if gotQueries[2].Get("type") != "open-jdk" {
		t.Errorf("expected type=open-jdk on remote call, got %q", gotQueries[2].Get("type"))
	}
}

func TestPipelinePluginRequestParamOverride(t *testing.T) {
	var mu sync.Mutex
	var gotQueries []url.Values
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotQueries = append(gotQueries, r.URL.Query())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		if strings.Contains(r.URL.Path, "/plugins/scheme") {
			_, _ = w.Write(jsonBody(t, map[string]giteego.PluginSchemeVO{
				"gcc-build@v1.0.0": {
					Type: giteego.PluginSchemeTypeVO{JSON: "gcc-build@v1.0.0", YAML: "build@gcc"},
					Config: []giteego.PluginComponentScheme{
						{
							Identifier: "PLUGIN_GCC_VERSION",
							Type:       "RemoteSelect",
							URL:        &giteego.RemoteSelectURLVO{GitOps: "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version?type=gcc"},
							Params:     map[string]string{"type": "gcc", "uuid": "${certificate}"},
						},
					},
				},
			}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.RemoteSelectResponse{Total: 1, List: []giteego.ComponentOptionVO{{Key: "9.4", Label: "GCC 9.4", Value: "9.4"}}}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "request", "PLUGIN_GCC_VERSION",
		"-R", "owner/repo", "--type", "gcc-build@v1.0.0",
		"-p", "certificate=uuid-123", "-p", "search=gcc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GCC 9.4") {
		t.Errorf("unexpected output: %s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotQueries) != 3 {
		t.Fatalf("expected 3 requests (billing+scheme+remote), got %v", gotQueries)
	}
	// ${certificate} in url.params is substituted from -p; literal types kept;
	// -p search added on top.
	if gotQueries[2].Get("uuid") != "uuid-123" {
		t.Errorf("expected uuid=uuid-123 (substituted), got %q", gotQueries[2].Get("uuid"))
	}
	if gotQueries[2].Get("type") != "gcc" {
		t.Errorf("expected type=gcc, got %q", gotQueries[2].Get("type"))
	}
	if gotQueries[2].Get("search") != "gcc" {
		t.Errorf("expected search=gcc, got %q", gotQueries[2].Get("search"))
	}
}
func TestPipelineBuildView(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotPathBase string
	f := newTestFactory(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "billing") {
			_, _ = w.Write(jsonBody(t, map[string]interface{}{"enabled": true}))
			return
		}
		_, _ = w.Write(jsonBody(t, giteego.PipelineBuildVO{
			ID:          1079,
			BuildNumber: 8,
			Status:      "FAILED",
			FileName:    "pipeline-20260622.yml",
			Ref:         "master",
			StartTime:   mustTime(t, "2026-06-24 02:05:00"),
			EndTime:     mustTime(t, "2026-06-24 03:19:13"),
			Trigger: &giteego.BuildTriggerVO{
				Username: "gitee",
				Data:     &giteego.TriggerData{Type: "WEBHOOK"},
			},
			Param: &giteego.BuildParamSetVO{
				InParams: []giteego.ParamValueVO{
					{Key: "GITEE_BRANCH", Value: "master"},
				},
			},
			Stages: []giteego.PipelineStageBuildVO{
				{
					Name:      "未命名",
					Status:    "FAILED",
					StartTime: mustTime(t, "2026-06-24 02:05:00"),
					EndTime:   mustTime(t, "2026-06-24 03:19:13"),
					Jobs: [][]giteego.PipelineJobBuildVO{
						{
							{
								Name:   "Maven 单元测试",
								Type:   "maven-unit-test@v1.0.0",
								Status: "FAILED",
								Data: &giteego.JobRunDataVO{
									Parameters: []giteego.ParamValueVO{
										{Key: "PLUGIN_JAVA_VERSION", Value: "8"},
									},
								},
								Record: &giteego.JobRecordVO{
									Status: "failure",
									Loggers: []giteego.JobLoggerVO{
										{Name: "任务执行", Status: "FAILED", Logger: "https://logs.example/1"},
										{Name: "缓存上传", Status: "FAILED", Logger: "https://logs.example/2"},
									},
								},
							},
						},
					},
				},
			},
		}))
	}, &gotPathBase)

	out, _, err := runPipelineCmd(t, f, "build", "view", "1079", "-R", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	// Trigger + duration (locale-agnostic values).
	if !strings.Contains(out, "WEBHOOK  by gitee") {
		t.Errorf("expected trigger line, got:\n%s", out)
	}
	if !strings.Contains(out, "74m13s") {
		t.Errorf("expected duration line, got:\n%s", out)
	}
	// Pipeline params section.
	if !strings.Contains(out, "GITEE_BRANCH = master") {
		t.Errorf("expected pipeline params, got:\n%s", out)
	}
	// Stage + job with params + loggers.
	if !strings.Contains(out, "未命名  [FAILED]") {
		t.Errorf("expected stage line, got:\n%s", out)
	}
	if !strings.Contains(out, "PLUGIN_JAVA_VERSION = 8") {
		t.Errorf("expected job param, got:\n%s", out)
	}
	if !strings.Contains(out, "https://logs.example/1") {
		t.Errorf("expected job loggers, got:\n%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests (billing+view), got %v", gotPaths)
	}
	if !strings.Contains(gotPaths[1], "/owner/repo/gitee-go/ipipe/rest/v5/pipelines/builds/1079") {
		t.Errorf("expected build view path, got %s", gotPaths[1])
	}
}

func mustTime(t *testing.T, value string) *giteego.FlexTime {
	t.Helper()
	tm, err := time.ParseInLocation(time.RFC3339, value, time.Local)
	if err != nil {
		// Fall back to the datetime layout the CLI displays.
		tm, err = time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		if err != nil {
			t.Fatalf("parse time %q: %v", value, err)
		}
	}
	return &giteego.FlexTime{Time: tm}
}
