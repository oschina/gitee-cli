package giteego

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRemoteSelectRoute(t *testing.T) {
	cases := []struct {
		path, serv, want string
	}{
		{
			path: "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version?type=gcc",
			serv: "/ipipe",
			want: "/rest/v5/external/remote-select/plugin-tools/version",
		},
		{
			path: "/gitee-go/sa/rest/v5/foo",
			serv: "/sa",
			want: "/rest/v5/foo",
		},
		{
			path: "/gitee-go/ipipe/rest/v5/plugins",
			serv: "/ipipe",
			want: "/rest/v5/plugins",
		},
		{
			path: "/rest/v5/external/remote-select/foo",
			serv: "/ipipe",
			want: "/rest/v5/external/remote-select/foo",
		},
	}
	for _, c := range cases {
		got := remoteSelectRoute(c.path, c.serv)
		if got != c.want {
			t.Errorf("remoteSelectRoute(%q, %q) = %q, want %q", c.path, c.serv, got, c.want)
		}
	}
}

func TestRemoteSelectQueryMerge(t *testing.T) {
	// The scheme URL already carries ?type=gcc; caller query overrides.
	got, err := remoteSelectQuery(
		"/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version?type=gcc&page=1",
		map[string]string{"page": "2"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got["type"] != "gcc" {
		t.Errorf("expected embedded type=gcc preserved, got %q", got["type"])
	}
	if got["page"] != "2" {
		t.Errorf("expected caller page=2 to override, got %q", got["page"])
	}
}

// TestRemoteSelectOptionsObjectForm decodes the {total,list} object form.
func TestRemoteSelectOptionsObjectForm(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{
			"total": 2,
			"list": []any{
				map[string]any{"key": "gcc", "label": "GCC", "value": "gcc"},
				map[string]any{"key": "go", "label": "Go", "value": "go"},
			},
		}))
	}, func(c *Client) error {
		resp, err := c.RemoteSelectOptions(context.Background(),
			"/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version?type=gcc", nil)
		if err != nil {
			return err
		}
		if resp.Total != 2 || len(resp.List) != 2 || resp.List[0].Value != "gcc" {
			t.Errorf("resp = %+v", resp)
		}
		return nil
	})
	want := "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version"
	if rr.path != want {
		t.Errorf("path = %q, want %q", rr.path, want)
	}
	if rr.query.Get("type") != "gcc" {
		t.Errorf("type = %q, want gcc", rr.query.Get("type"))
	}
}

// TestRemoteSelectOptionsArrayForm falls back to the bare-array form.
func TestRemoteSelectOptionsArrayForm(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, []any{map[string]any{"key": "go", "value": "go"}}))
	}, func(c *Client) error {
		resp, err := c.RemoteSelectOptions(context.Background(),
			"/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version", nil)
		if err != nil {
			return err
		}
		if resp.Total != 1 || len(resp.List) != 1 {
			t.Errorf("resp = %+v, want array fallback total 1", resp)
		}
		return nil
	})
	if rr.path != "/gitee-go/ipipe/rest/v5/external/remote-select/plugin-tools/version" {
		t.Errorf("path = %q", rr.path)
	}
}

// TestListPlugins asserts the gitops plugins route and category decode.
func TestListPlugins(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, []CategoryPluginVO{{
			Name:    "构建",
			Plugins: []PluginVO{{Name: "maven-build", Type: "MAVEN_JOB"}},
		}}))
	}, func(c *Client) error {
		cats, err := c.ListPlugins(context.Background())
		if err != nil {
			return err
		}
		if len(cats) != 1 || len(cats[0].Plugins) != 1 || cats[0].Plugins[0].Type != "MAVEN_JOB" {
			t.Errorf("categories = %+v", cats)
		}
		return nil
	})
	if rr.method != http.MethodGet || rr.path != "/gitee-go/ipipe/rest/v5/plugins" {
		t.Errorf("got %s %s, want GET /gitee-go/ipipe/rest/v5/plugins", rr.method, rr.path)
	}
}

// TestGetPluginSchemeResolvesByJobType returns the scheme keyed by jobType.
func TestGetPluginSchemeResolvesByJobType(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{
			"MAVEN_JOB": map[string]any{"type": map[string]any{"json": "MAVEN_JOB"}},
			"SHELL_JOB": map[string]any{"type": map[string]any{"json": "SHELL_JOB"}},
		}))
	})
	scheme, err := c.GetPluginScheme(context.Background(), "SHELL_JOB")
	if err != nil {
		t.Fatal(err)
	}
	if scheme == nil || scheme.Type.JSON != "SHELL_JOB" {
		t.Errorf("scheme = %+v, want SHELL_JOB", scheme)
	}
}

// TestGetPluginSchemeFirstFallback returns the first entry when jobType is absent.
func TestGetPluginSchemeFirstFallback(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, map[string]any{
			"MAVEN_JOB": map[string]any{"type": map[string]any{"json": "MAVEN_JOB"}},
		}))
	})
	scheme, err := c.GetPluginScheme(context.Background(), "NOPE_JOB")
	if err != nil {
		t.Fatal(err)
	}
	if scheme == nil || scheme.Type.JSON != "MAVEN_JOB" {
		t.Errorf("scheme = %+v, want first entry MAVEN_JOB", scheme)
	}
}

// TestGetPluginSchemeEmpty returns nil for an empty scheme set.
func TestGetPluginSchemeEmpty(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, nil))
	})
	scheme, err := c.GetPluginScheme(context.Background(), "NOPE_JOB")
	if err != nil {
		t.Fatal(err)
	}
	if scheme != nil {
		t.Errorf("scheme = %+v, want nil", scheme)
	}
}

// TestGeneratePluginExampleYaml asserts the jobType query and YAML decode.
func TestGeneratePluginExampleYaml(t *testing.T) {
	rr := runAndRecord(t, "tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, jsonBody(t, "- step: build\n  job: MAVEN_JOB"))
	}, func(c *Client) error {
		yaml, err := c.GeneratePluginExampleYaml(context.Background(), "MAVEN_JOB")
		if err != nil {
			return err
		}
		if yaml != "- step: build\n  job: MAVEN_JOB" {
			t.Errorf("yaml = %q", yaml)
		}
		return nil
	})
	if rr.method != http.MethodGet || rr.path != "/gitee-go/ipipe/rest/v5/plugins/example" {
		t.Errorf("got %s %s, want GET /gitee-go/ipipe/rest/v5/plugins/example", rr.method, rr.path)
	}
	if rr.query.Get("jobType") != "MAVEN_JOB" {
		t.Errorf("jobType = %q, want MAVEN_JOB", rr.query.Get("jobType"))
	}
}

// TestPluginCodeNonZeroError verifies code!=0 surfaces the msg.
func TestPluginCodeNonZeroError(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":500,"msg":"plugin backend down","data":null}`)
	})
	if _, err := c.ListPlugins(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "plugin backend down") {
		t.Errorf("expected code!=0 error, got %v", err)
	}
}
