package update

import (
	"context"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPathWithin(t *testing.T) {
	root := filepath.Join("tmp", "node_modules")
	if !pathWithin(root, filepath.Join(root, "@gitee", "gitee-cli")) {
		t.Fatal("expected package path to be within npm root")
	}
	if pathWithin(root, filepath.Join("tmp", "other")) {
		t.Fatal("unexpected path match")
	}
}

func TestReleaseArchiveName(t *testing.T) {
	name := releaseArchiveName("v1.2.3")
	wantSuffix := ".tar.gz"
	if runtime.GOOS == "windows" {
		wantSuffix = ".zip"
	}
	if !strings.Contains(name, "gitee_1.2.3_"+runtime.GOOS+"_"+runtime.GOARCH) || !strings.HasSuffix(name, wantSuffix) {
		t.Fatalf("unexpected archive name: %s", name)
	}
}

func TestApplyUpdateRejectsLocalNPMInstall(t *testing.T) {
	_, err := ApplyUpdate(context.Background(), &ReleaseInfo{Version: "v1.2.3"}, Installation{Method: InstallNPM, ScopeKnown: true}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "local npm installation") {
		t.Fatalf("unexpected error: %v", err)
	}
}
