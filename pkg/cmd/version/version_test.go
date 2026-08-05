package version

import (
	"bytes"
	"io"
	"runtime"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func newVersionTestFactory() (*cmdutil.Factory, *bytes.Buffer) {
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

func TestVersionCmd_plainText(t *testing.T) {
	f, outBuf := newVersionTestFactory()
	cmd := NewVersionCmd(f)
	cmd.SetOut(outBuf)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := outBuf.String()
	if !strings.Contains(out, "Gitee CLI") {
		t.Errorf("expected 'Gitee CLI' header in output, got: %s", out)
	}
	if !strings.Contains(out, "Version") {
		t.Errorf("expected 'Version' field in output, got: %s", out)
	}
	if !strings.Contains(out, runtime.Version()) {
		t.Errorf("expected go version in output, got: %s", out)
	}
	if !strings.Contains(out, runtime.GOOS) {
		t.Errorf("expected OS in output, got: %s", out)
	}
	if !strings.Contains(out, repositoryURL) {
		t.Errorf("expected repository URL in output, got: %s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("plain output contains ANSI escape sequences: %q", out)
	}
}

func TestPrintFancy_includesRepositoryURL(t *testing.T) {
	f, outBuf := newVersionTestFactory()
	if err := printFancy(f); err != nil {
		t.Fatal(err)
	}
	out := outBuf.String()
	if !strings.Contains(out, "GITEE CLI") {
		t.Errorf("expected compact product title, got: %s", out)
	}
	if !strings.Contains(out, repositoryURL) {
		t.Errorf("expected repository URL in output, got: %s", out)
	}
}
