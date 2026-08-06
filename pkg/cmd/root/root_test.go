package root

import (
	"bytes"
	"context"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func resetConfig(t *testing.T) {
	t.Helper()
	config.Reset()
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	_ = config.Load()
}

func newTestFactory() *cmdutil.Factory {
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	return &cmdutil.Factory{
		IOStreams: ios,
		GiteeClient: func() (*gitee.Client, error) {
			return gitee.NewClient("test-token"), nil
		},
	}
}

func setAlias(t *testing.T, name, expansion string) {
	t.Helper()
	if err := config.Set("alias."+name, expansion); err != nil {
		t.Fatalf("set alias: %v", err)
	}
}

func TestExpandShellAlias_basic(t *testing.T) {
	resetConfig(t)
	setAlias(t, "hello", "!echo hello")
	_ = config.Load()

	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	script, ok := expandShellAlias(rootCmd, []string{"hello"})
	if !ok {
		t.Fatal("expected shell alias to be detected")
	}
	if script != "echo hello" {
		t.Errorf("unexpected script: %q", script)
	}
}

func TestExpandShellAlias_positionalVars(t *testing.T) {
	resetConfig(t)
	setAlias(t, "greet", "!echo $1")
	_ = config.Load()

	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	script, ok := expandShellAlias(rootCmd, []string{"greet", "world"})
	if !ok {
		t.Fatal("expected shell alias to be detected")
	}
	if script != "echo world" {
		t.Errorf("unexpected script: %q", script)
	}
}

func TestExpandShellAlias_appendExtraArgs(t *testing.T) {
	resetConfig(t)
	setAlias(t, "run", "!make")
	_ = config.Load()

	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	script, ok := expandShellAlias(rootCmd, []string{"run", "build", "test"})
	if !ok {
		t.Fatal("expected shell alias to be detected")
	}
	if script != "make build test" {
		t.Errorf("unexpected script: %q", script)
	}
}

func TestExpandShellAlias_notShellAlias(t *testing.T) {
	resetConfig(t)
	setAlias(t, "prs", "pr list -s open")
	_ = config.Load()

	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	_, ok := expandShellAlias(rootCmd, []string{"prs"})
	if ok {
		t.Error("expected non-shell alias to be skipped")
	}
}

func TestExpandShellAlias_nativeCommandNotExpanded(t *testing.T) {
	resetConfig(t)
	setAlias(t, "pr", "!echo hijacked")
	_ = config.Load()

	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	_, ok := expandShellAlias(rootCmd, []string{"pr", "list"})
	if ok {
		t.Error("native command should not be expanded as shell alias")
	}
}

func TestExpandAlias_shellAliasSkipped(t *testing.T) {
	resetConfig(t)
	setAlias(t, "deploy", "!gitee pr create")
	_ = config.Load()

	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	_, ok := expandAlias(rootCmd, []string{"deploy"})
	if ok {
		t.Error("shell alias should not be handled by expandAlias")
	}
}

func TestRunShellAliasUsesFactoryIOStreams(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(strings.NewReader("")),
		Out:    out,
		ErrOut: errOut,
	}
	script := "printf agent-output"
	if runtime.GOOS == "windows" {
		script = "echo agent-output"
	}

	if code := runShellAlias(script, ios); code != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "agent-output") {
		t.Fatalf("expected shell output in factory stream, got %q", out.String())
	}
}

func TestLeafCommandsRejectUnexpectedArguments(t *testing.T) {
	testCases := [][]string{
		{"ai", "first", "second"},
		{"alias", "list", "extra"},
		{"auth", "login", "extra"},
		{"auth", "logout", "extra"},
		{"auth", "status", "extra"},
		{"auth", "token", "extra"},
		{"config", "extra"},
		{"config", "list", "extra"},
		{"gist", "list", "extra"},
		{"issue", "list", "extra"},
		{"issue", "create", "extra"},
		{"pr", "list", "extra"},
		{"pr", "create", "extra"},
		{"release", "list", "extra"},
		{"release", "create", "extra"},
		{"repo", "list", "extra"},
		{"repo", "create", "extra"},
		{"update", "extra"},
		{"ssh-key", "list", "extra"},
		{"ssh-key", "add", "extra"},
		{"version", "extra"},
	}

	for _, args := range testCases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			rootCmd := NewRootCmd(context.Background(), newTestFactory())
			rootCmd.SetArgs(args)
			if err := rootCmd.Execute(); err == nil {
				t.Fatalf("expected %q to reject unexpected arguments", strings.Join(args, " "))
			}
		})
	}
}

func TestAllCommandsRenderHelp(t *testing.T) {
	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	var visit func(*cobra.Command)
	visit = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			var out bytes.Buffer
			child.SetOut(&out)
			if err := child.Help(); err != nil {
				t.Errorf("%s help failed: %v", child.CommandPath(), err)
			} else if !strings.Contains(out.String(), "Usage:") {
				t.Errorf("%s help did not contain usage text", child.CommandPath())
			}
			visit(child)
		}
	}
	visit(rootCmd)
}

func TestRootRegistersSearchInsteadOfUser(t *testing.T) {
	rootCmd := NewRootCmd(context.Background(), newTestFactory())
	if command, _, err := rootCmd.Find([]string{"search", "repos"}); err != nil || command.Name() != "repos" {
		t.Fatalf("search repos command not registered: command=%v err=%v", command, err)
	}
	for _, command := range rootCmd.Commands() {
		if command.Name() == "user" {
			t.Fatal("legacy user command is still registered")
		}
	}
}

func TestHasQuietFlag(t *testing.T) {
	for _, args := range [][]string{{"pr", "list", "--quiet"}, {"-q", "version"}, {"pr", "list", "-Vq"}} {
		if !hasQuietFlag(args) {
			t.Errorf("expected quiet flag in %v", args)
		}
	}
	if hasQuietFlag([]string{"pr", "list"}) {
		t.Error("unexpected quiet flag")
	}
}

func TestSkipUpdateForArgs(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"version"}, {"update"}, {"completion", "zsh"}, {"pr", "list", "-h"}} {
		if !skipUpdateForArgs(args) {
			t.Errorf("expected update check to be skipped for %v", args)
		}
	}
	if skipUpdateForArgs([]string{"pr", "list"}) {
		t.Error("unexpected update skip for regular command")
	}
}
