//go:build !windows

package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "gitee")
	replacement := filepath.Join(dir, "replacement")
	if err := os.WriteFile(current, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(current, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("new"), 0700); err != nil {
		t.Fatal(err)
	}
	deferred, err := replaceExecutable(current, replacement)
	if err != nil {
		t.Fatal(err)
	}
	if deferred {
		t.Fatal("Unix replacement should not be deferred")
	}
	data, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("updated executable contains %q", data)
	}
	info, err := os.Stat(current)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("mode = %o, want 755", info.Mode().Perm())
	}
}
