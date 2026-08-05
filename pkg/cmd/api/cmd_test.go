package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/build"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func runAPICmd(args []string, handler http.Handler) (string, string, error) {
	srv := httptest.NewServer(handler)
	defer srv.Close()

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    outBuf,
		ErrOut: errBuf,
	}
	f := &cmdutil.Factory{
		IOStreams: ios,
		GiteeClient: func() (*gitee.Client, error) {
			return gitee.NewClient("test-token", gitee.WithBaseURL(srv.URL)), nil
		},
	}

	t := testing.T{}
	t.Setenv("GITEE_TOKEN", "test-token")
	t.Setenv("GITEE_API_PREFIX", srv.URL)

	cmd := NewAPICmd(f)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func runAPICmdWithEnv(t *testing.T, args []string, handler http.Handler) (string, string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	t.Setenv("GITEE_TOKEN", "test-token")
	t.Setenv("GITEE_API_PREFIX", srv.URL)

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    outBuf,
		ErrOut: errBuf,
	}
	f := &cmdutil.Factory{
		IOStreams: ios,
		GiteeClient: func() (*gitee.Client, error) {
			return gitee.NewClient("test-token", gitee.WithBaseURL(srv.URL)), nil
		},
	}

	cmd := NewAPICmd(f)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestNewAPICmd_get(t *testing.T) {
	payload := map[string]string{"login": "alice"}
	out, _, err := runAPICmdWithEnv(t, []string{"/user"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("User-Agent"); got != build.UserAgent() {
			t.Errorf("expected %q user agent, got %q", build.UserAgent(), got)
		}
		json.NewEncoder(w).Encode(payload)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected 'alice' in output, got: %s", out)
	}
}

func TestNewAPICmd_post_with_fields(t *testing.T) {
	var capturedBody map[string]string
	out, _, err := runAPICmdWithEnv(t, []string{"/repos/owner/repo/issues", "-X", "POST", "-f", "title=Bug", "-f", "body=desc"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			json.NewDecoder(r.Body).Decode(&capturedBody)
			json.NewEncoder(w).Encode(map[string]interface{}{"number": "I1"})
		}))
	if err != nil {
		t.Fatal(err)
	}
	if capturedBody["title"] != "Bug" {
		t.Errorf("expected title=Bug in body, got: %v", capturedBody)
	}
	if !strings.Contains(out, "I1") {
		t.Errorf("expected issue number in output, got: %s", out)
	}
}

func TestNewAPICmd_raw_body(t *testing.T) {
	rawJSON := `{"state":"closed"}`
	var capturedBody map[string]string
	_, _, err := runAPICmdWithEnv(t, []string{"/repos/o/r/issues/1", "-X", "PATCH", "--body", rawJSON},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			json.NewEncoder(w).Encode(map[string]string{"state": "closed"})
		}))
	if err != nil {
		t.Fatal(err)
	}
	if capturedBody["state"] != "closed" {
		t.Errorf("expected state=closed, got: %v", capturedBody)
	}
}

func TestNewAPICmd_custom_header(t *testing.T) {
	var capturedHeader string
	_, _, err := runAPICmdWithEnv(t, []string{"/user", "-H", "X-Custom: myvalue"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeader = r.Header.Get("X-Custom")
			json.NewEncoder(w).Encode(map[string]string{})
		}))
	if err != nil {
		t.Fatal(err)
	}
	if capturedHeader != "myvalue" {
		t.Errorf("expected X-Custom=myvalue, got: %q", capturedHeader)
	}
}

func TestNewAPICmd_error_status(t *testing.T) {
	_, errOut, _ := runAPICmdWithEnv(t, []string{"/notfound"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"not found"}`))
		}))
	if !strings.Contains(errOut, "404") {
		t.Errorf("expected 404 in stderr, got: %s", errOut)
	}
}

func TestNewAPICmd_requires_arg(t *testing.T) {
	_, _, err := runAPICmdWithEnv(t, []string{}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error when no endpoint provided")
	}
}

func TestNewAPICmd_invalid_header(t *testing.T) {
	_, _, err := runAPICmdWithEnv(t, []string{"/user", "-H", "BadHeader"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{})
		}))
	if err == nil {
		t.Error("expected error for invalid header format")
	}
}

func TestNewAPICmd_pretty_json(t *testing.T) {
	out, _, err := runAPICmdWithEnv(t, []string{"/user", "-p"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"login":"alice","name":"Alice"}`))
		}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n") || !strings.Contains(out, "alice") {
		t.Errorf("expected pretty-printed JSON with newlines, got: %s", out)
	}
}
