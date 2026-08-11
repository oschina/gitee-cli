package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func TestAuthStatusCmd_notLoggedIn(t *testing.T) {
	t.Setenv("GITEE_TOKEN", "")
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())

	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()

	tf.Factory.GiteeClient = func() (*gitee.Client, error) {
		return nil, config.ErrNotLoggedIn
	}

	root := NewAuthCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs([]string{"status"})
	_ = root.Execute()

	out := tf.Output()
	if !strings.Contains(out, "not logged in") && !strings.Contains(out, "未登录") {
		t.Errorf("expected not-logged-in message, got: %s", out)
		t.Errorf("expected 'not logged in' in output, got: %s", out)
	}
}

func TestAuthStatusCmd_multiHost(t *testing.T) {
	user := gitee.User{Login: "carol", Name: "Carol"}
	out, err := runAuthCmd([]string{"status"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(user)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "carol") {
		t.Errorf("expected username in output, got: %s", out)
	}
}

func TestAuthLogoutCmd(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	_ = config.Load()

	if err := config.SaveToken("dummy-token"); err != nil {
		t.Fatal(err)
	}

	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()
	root := NewAuthCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs([]string{"logout"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := tf.Output()
	if !strings.Contains(out, "Logged out") && !strings.Contains(out, "已退出登录") {
		t.Errorf("expected logout message, got: %s", out)
	}
}

func TestAuthLoginCmd_withTokenFromStdin(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	user := gitee.User{Login: "agent", Name: "Automation Agent"}

	tf := cmdtest.NewTestFactory(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); !strings.Contains(got, "stdin-token") {
			t.Errorf("expected stdin token in authorization header, got %q", got)
		}
		json.NewEncoder(w).Encode(user)
	}))
	defer tf.Close()
	tf.IOStreams.In = io.NopCloser(strings.NewReader("stdin-token\n"))
	t.Setenv("GITEE_API_PREFIX", tf.Server.URL)

	root := NewAuthCmd(tf.Factory)
	root.SetArgs([]string{"login", "--with-token"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tf.Output(), "agent") {
		t.Fatalf("expected login confirmation, got %q", tf.Output())
	}
}

func TestAuthLoginCmd_setsCustomHostAsDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	user := gitee.User{Login: "agent", Name: "Automation Agent"}
	tf := cmdtest.NewTestFactory(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(user)
	}))
	defer tf.Close()
	tf.IOStreams.In = io.NopCloser(strings.NewReader("private-token\n"))

	const hostname = "git.example.com"
	if err := config.SaveHostConfig(hostname, "old-token", tf.Server.URL); err != nil {
		t.Fatal(err)
	}

	root := NewAuthCmd(tf.Factory)
	root.SetArgs([]string{"login", "--hostname", hostname, "--with-token"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := config.Get(config.KeyHost); got != hostname {
		t.Fatalf("expected default host %q, got %q", hostname, got)
	}
	hc, ok := config.GetHostConfig(hostname)
	if !ok || hc.Token != "private-token" {
		t.Fatalf("expected updated private host credentials, got %#v", hc)
	}
}

func TestAuthLogoutCmd_resetsCustomDefaultHost(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	if err := config.Load(); err != nil {
		t.Fatal(err)
	}
	const hostname = "git.example.com"
	if err := config.SaveHostConfig(hostname, "token", ""); err != nil {
		t.Fatal(err)
	}
	if err := config.Set(config.KeyHost, hostname); err != nil {
		t.Fatal(err)
	}

	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()
	tf.Factory.Hostname = hostname
	root := NewAuthCmd(tf.Factory)
	root.SetArgs([]string{"logout"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := config.Get(config.KeyHost); got != config.DefaultHost {
		t.Fatalf("expected default host reset to %q, got %q", config.DefaultHost, got)
	}
}

func TestAuthLoginCmd_rejectsTokenFlag(t *testing.T) {
	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()

	root := NewAuthCmd(tf.Factory)
	root.SetArgs([]string{"login", "--token", "arg-token"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown flag: --token") {
		t.Fatalf("expected command-line token flag to be rejected, got %v", err)
	}
}
