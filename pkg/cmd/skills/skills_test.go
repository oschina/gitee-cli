package skills

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
	skillassets "gitee.com/oschina/gitee-cli/skills"
)

func runSkillsCmd(args ...string) (string, error) {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &iostreams.IOStreams{
			In:     io.NopCloser(bytes.NewReader(nil)),
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}
	cmd := NewSkillsCmd(f)
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestInstallListAndReinstall(t *testing.T) {
	target := t.TempDir()
	for _, legacy := range legacySkillNames {
		if err := os.MkdirAll(filepath.Join(target, legacy), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	unrelated := filepath.Join(target, "third-party-skill")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runSkillsCmd("--dir", target, "install", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result changeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("decode install JSON: %v\n%s", err, out)
	}
	if len(result.Installed) != 5 {
		t.Fatalf("installed = %v", result.Installed)
	}
	if len(result.RemovedLegacy) != len(legacySkillNames) {
		t.Fatalf("removed legacy = %v", result.RemovedLegacy)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated skill was changed: %v", err)
	}

	names, err := skillassets.Names()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		want, err := fs.ReadFile(skillassets.Files, name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(target, name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("installed %s does not match embedded content", name)
		}
	}

	stale := filepath.Join(target, "gitee-api", "stale.txt")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runSkillsCmd("--dir", target, "install"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("mirror install left stale file: %v", err)
	}

	out, err = runSkillsCmd("--dir", target, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var statuses []skillStatus
	if err := json.Unmarshal([]byte(out), &statuses); err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 5 {
		t.Fatalf("statuses = %v", statuses)
	}
	for _, status := range statuses {
		if !status.Installed {
			t.Fatalf("expected %s to be installed", status.Name)
		}
	}
}

func TestUninstallRequiresYesOutsideTerminal(t *testing.T) {
	target := t.TempDir()
	if _, err := runSkillsCmd("--dir", target, "uninstall"); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected --yes error, got %v", err)
	}
}

func TestPlainOutputIsFormatted(t *testing.T) {
	target := t.TempDir()
	out, err := runSkillsCmd("--dir", target, "install")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "%!") || !strings.Contains(out, target) {
		t.Fatalf("unexpected install output: %q", out)
	}
	out, err = runSkillsCmd("--dir", target, "uninstall", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "%!") || !strings.Contains(out, target) {
		t.Fatalf("unexpected uninstall output: %q", out)
	}
}

func TestUninstallRemovesOnlyManagedSkills(t *testing.T) {
	target := t.TempDir()
	if _, err := runSkillsCmd("--dir", target, "install"); err != nil {
		t.Fatal(err)
	}
	for _, legacy := range legacySkillNames {
		if err := os.MkdirAll(filepath.Join(target, legacy), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	unrelated := filepath.Join(target, "third-party-skill")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runSkillsCmd("--dir", target, "uninstall", "--yes", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result changeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Removed) != 7 {
		t.Fatalf("removed = %v", result.Removed)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated skill was removed: %v", err)
	}
	for _, name := range result.Removed {
		if _, err := os.Stat(filepath.Join(target, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still exists: %v", name, err)
		}
	}
}

func TestResolveTargetDirUsesEnvironment(t *testing.T) {
	target := t.TempDir()
	t.Setenv(skillsDirEnv, target)
	got, err := resolveTargetDir("")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
}
