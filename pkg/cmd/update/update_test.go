package update

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/build"
	internalupdate "gitee.com/oschina/gitee-cli/internal/update"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func TestUpdateCheckOnly(t *testing.T) {
	called := false
	out, err := runUpdateCmd(t, []string{"--check"}, commandDependencies{
		check: func(version string) (*internalupdate.ReleaseInfo, error) {
			return &internalupdate.ReleaseInfo{Version: "v2.0.0", URL: "https://example.com/v2.0.0"}, nil
		},
		detect: func(context.Context) (internalupdate.Installation, error) {
			return internalupdate.Installation{Method: internalupdate.InstallStandalone, Detail: "standalone binary"}, nil
		},
		apply: func(context.Context, *internalupdate.ReleaseInfo, internalupdate.Installation, io.Writer, io.Writer) (internalupdate.InstallResult, error) {
			called = true
			return internalupdate.InstallResult{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("--check must not install")
	}
	if !strings.Contains(out, "Latest version:   v2.0.0") || !strings.Contains(out, "standalone binary") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestUpdateYesAppliesUpdate(t *testing.T) {
	called := false
	out, err := runUpdateCmd(t, []string{"--yes"}, commandDependencies{
		check: func(string) (*internalupdate.ReleaseInfo, error) {
			return &internalupdate.ReleaseInfo{Version: "v2.0.0"}, nil
		},
		detect: func(context.Context) (internalupdate.Installation, error) {
			return internalupdate.Installation{Method: internalupdate.InstallNPM, Global: true, Detail: "npm global"}, nil
		},
		apply: func(_ context.Context, release *internalupdate.ReleaseInfo, installation internalupdate.Installation, _, _ io.Writer) (internalupdate.InstallResult, error) {
			called = release.Version == "v2.0.0" && installation.Method == internalupdate.InstallNPM
			return internalupdate.InstallResult{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called || !strings.Contains(out, "Updated gitee-cli to v2.0.0") {
		t.Fatalf("update not applied, output: %s", out)
	}
}

func TestUpdateRequiresYesWithoutTerminal(t *testing.T) {
	_, err := runUpdateCmd(t, nil, commandDependencies{
		check: func(string) (*internalupdate.ReleaseInfo, error) {
			return &internalupdate.ReleaseInfo{Version: "v2.0.0"}, nil
		},
		detect: func(context.Context) (internalupdate.Installation, error) {
			return internalupdate.Installation{Method: internalupdate.InstallStandalone}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "--yes is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func runUpdateCmd(t *testing.T, args []string, dependencies commandDependencies) (string, error) {
	t.Helper()
	originalVersion := build.Version
	build.Version = "v1.0.0"
	t.Cleanup(func() { build.Version = originalVersion })
	out := &bytes.Buffer{}
	factory := &cmdutil.Factory{
		Context: context.Background(),
		IOStreams: &iostreams.IOStreams{
			In: io.NopCloser(strings.NewReader("")), Out: out, ErrOut: &bytes.Buffer{},
		},
	}
	if dependencies.pending == nil {
		dependencies.pending = func(string) error { return nil }
	}
	cmd := newUpdateCmd(factory, dependencies)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
