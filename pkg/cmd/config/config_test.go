package config

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/config"
	cfg "gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func newConfigTestFactory() (*cmdutil.Factory, *bytes.Buffer) {
	outBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    outBuf,
		ErrOut: &bytes.Buffer{},
	}
	f := &cmdutil.Factory{
		IOStreams: ios,
		GiteeClient: func() (*gitee.Client, error) {
			return gitee.NewClient("test-token"), nil
		},
	}
	return f, outBuf
}

func runConfigCmd(t *testing.T, args []string) (string, error) {
	t.Helper()
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	_ = config.Load()

	f, outBuf := newConfigTestFactory()
	root := NewConfigCmd(f)
	root.SetOut(outBuf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.Execute()
	return outBuf.String(), err
}

func TestConfigSetAndGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	_ = cfg.Load()

	f, outBuf := newConfigTestFactory()
	root := NewConfigCmd(f)
	root.SetOut(outBuf)
	root.SetErr(&bytes.Buffer{})

	root.SetArgs([]string{"set", "editor", "vim"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	setOut := outBuf.String()
	if !strings.Contains(setOut, "editor") || !strings.Contains(setOut, "vim") {
		t.Errorf("expected set confirmation, got: %s", setOut)
	}

	outBuf.Reset()
	root.SetArgs([]string{"get", "editor"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(outBuf.String(), "vim") {
		t.Errorf("expected 'vim' from get, got: %s", outBuf.String())
	}
}

func TestConfigGetCmd(t *testing.T) {
	t.Setenv("GITEE_API_PREFIX", "https://example.com/api/v5")
	out, err := runConfigCmd(t, []string{"get", "api_prefix"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "example.com") {
		t.Errorf("expected api_prefix in output, got: %s", out)
	}
}

func TestConfigSetCmd_requiresTwoArgs(t *testing.T) {
	_, err := runConfigCmd(t, []string{"set", "onlykey"})
	if err == nil {
		t.Error("expected error with only one argument to set")
	}
}

func TestConfigSetCmd_doesNotOpenTUIWithoutTerminal(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Set(cfg.KeyTUI, "true"); err != nil {
		t.Fatal(err)
	}

	f, _ := newConfigTestFactory()
	root := NewConfigCmd(f)
	root.SetArgs([]string{"set", cfg.KeyTheme})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "value required") {
		t.Fatalf("expected a value-required error instead of a TUI prompt, got %v", err)
	}
}

func TestConfigGetCmd_requiresOneArg(t *testing.T) {
	_, err := runConfigCmd(t, []string{"get"})
	if err == nil {
		t.Error("expected error with no argument to get")
	}
}

func TestConfigListCmd(t *testing.T) {
	out, err := runConfigCmd(t, []string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("expected some config output from list")
	}
}

func TestConfigListCmd_json(t *testing.T) {
	out, err := runConfigCmd(t, []string{"list", "-j"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "{") {
		t.Errorf("expected JSON output from list -j, got: %s", out)
	}
}

func TestSensitiveConfigOutputIsRedacted(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}

	f, outBuf := newConfigTestFactory()
	f.IOStreams.In = io.NopCloser(strings.NewReader("ai-secret\n"))
	root := NewConfigCmd(f)
	root.SetOut(outBuf)
	root.SetErr(&bytes.Buffer{})

	root.SetArgs([]string{"set", "AI.TOKEN", "--stdin"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if output := outBuf.String(); strings.Contains(output, "ai-secret") || !strings.Contains(output, cfg.RedactedValue) {
		t.Fatalf("set output was not redacted: %s", output)
	}

	outBuf.Reset()
	root.SetArgs([]string{"get", cfg.KeyAIToken})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if output := outBuf.String(); strings.Contains(output, "ai-secret") || !strings.Contains(output, cfg.RedactedValue) {
		t.Fatalf("get output was not redacted: %s", output)
	}

	outBuf.Reset()
	root.SetArgs([]string{"list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if output := outBuf.String(); strings.Contains(output, "ai-secret") || !strings.Contains(output, "redacted") {
		t.Fatalf("list output was not redacted: %s", output)
	}
}

func TestSensitiveConfigRejectsCommandLineValue(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}
	f, _ := newConfigTestFactory()
	root := NewConfigCmd(f)
	root.SetArgs([]string{"set", cfg.KeyAIToken, "secret"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--stdin") {
		t.Fatalf("expected command-line secret to be rejected, got %v", err)
	}
}

func TestSensitiveConfigRejectsEmptyStdin(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}
	f, _ := newConfigTestFactory()
	root := NewConfigCmd(f)
	root.SetArgs([]string{"set", cfg.KeyAIToken, "--stdin"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("expected empty stdin to be rejected, got %v", err)
	}
}
