package alias

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

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

func newAliasTestFactory() (*cmdutil.Factory, *bytes.Buffer) {
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

func runAliasCmd(args []string) (string, error) {
	f, outBuf := newAliasTestFactory()
	root := NewAliasCmd(f)
	root.SetOut(outBuf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.Execute()
	return outBuf.String(), err
}

func TestAliasSetAndList(t *testing.T) {
	resetConfig(t)

	out, err := runAliasCmd([]string{"set", "prs", "pr list -s open"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "prs") {
		t.Errorf("expected alias name in output, got: %s", out)
	}

	_ = config.Load()

	out, err = runAliasCmd([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "prs") {
		t.Errorf("expected 'prs' alias in list output, got: %s", out)
	}
	if !strings.Contains(out, "pr list -s open") {
		t.Errorf("expected expansion in list output, got: %s", out)
	}
}

func TestAliasSetCmd(t *testing.T) {
	resetConfig(t)

	out, err := runAliasCmd([]string{"set", "myalias", "repo list"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "myalias") {
		t.Errorf("expected alias name in output, got: %s", out)
	}
	if !strings.Contains(out, "repo list") {
		t.Errorf("expected expansion in output, got: %s", out)
	}
}

func TestAliasSetCmd_requiresTwoArgs(t *testing.T) {
	_, err := runAliasCmd([]string{"set", "onlyname"})
	if err == nil {
		t.Error("expected error with only one argument")
	}
}

func TestAliasDeleteCmd(t *testing.T) {
	resetConfig(t)

	_, err := runAliasCmd([]string{"set", "todel", "issue list"})
	if err != nil {
		t.Fatal(err)
	}
	_ = config.Load()

	out, err := runAliasCmd([]string{"delete", "todel"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "todel") {
		t.Errorf("expected alias name in delete output, got: %s", out)
	}
}

func TestAliasDeleteCmd_notFound(t *testing.T) {
	resetConfig(t)

	_, err := runAliasCmd([]string{"delete", "nonexistent"})
	if err == nil {
		t.Error("expected error when deleting non-existent alias")
	}
}

func TestAliasListCmd_empty(t *testing.T) {
	resetConfig(t)

	out, err := runAliasCmd([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No aliases configured") && !strings.Contains(out, "暂无别名配置") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestAliasListCmd_json(t *testing.T) {
	resetConfig(t)

	_, _ = runAliasCmd([]string{"set", "jsonalias", "gist list"})
	_ = config.Load()

	out, err := runAliasCmd([]string{"list", "-j"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "jsonalias") {
		t.Errorf("expected alias in JSON output, got: %s", out)
	}
}

func TestAliasListCmd_emptyJSON(t *testing.T) {
	resetConfig(t)

	out, err := runAliasCmd([]string{"list", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out); got != "{}" {
		t.Fatalf("expected an empty JSON object, got %q", got)
	}
}

// Silence unused import warning for net/http
var _ = http.NotFoundHandler
