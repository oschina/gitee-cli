package giteego

import (
	"strings"
	"testing"
)

// TestOpenURL builds the gitee_go open URL from a platform base URL.
func TestOpenURL(t *testing.T) {
	got, err := OpenURL("https://gitee.com/api/v5", "mr-chenguang", "test-gitee-go")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://gitee.com/mr-chenguang/test-gitee-go/gitee_go/open?qt=path"
	if got != want {
		t.Errorf("OpenURL = %q, want %q", got, want)
	}
}

// TestOpenURLIgnoresAPIPath asserts only scheme + host are used from the base.
func TestOpenURLIgnoresAPIPath(t *testing.T) {
	got, err := OpenURL("https://premium-k8s.gitee.cn/api/custom", "autodeploy", "app")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://premium-k8s.gitee.cn/autodeploy/app/gitee_go/open?qt=path"
	if got != want {
		t.Errorf("OpenURL = %q, want %q", got, want)
	}
}

// TestOpenURLEscapesPath assert owner/repo are PathEscaped.
func TestOpenURLEscapesPath(t *testing.T) {
	got, err := OpenURL("https://gitee.com/api/v5", "team/one", "my repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/team%2Fone/my%20repo/gitee_go/open") {
		t.Errorf("OpenURL = %q, want escaped owner/repo", got)
	}
}

// TestOpenURLEmptyOwnerRepo covers the missing owner/repo error.
func TestOpenURLEmptyOwnerRepo(t *testing.T) {
	if _, err := OpenURL("https://gitee.com/api/v5", "", "repo"); err == nil ||
		!strings.Contains(err.Error(), "owner and repo are required") {
		t.Errorf("expected owner error, got %v", err)
	}
	if _, err := OpenURL("https://gitee.com/api/v5", "owner", ""); err == nil ||
		!strings.Contains(err.Error(), "owner and repo are required") {
		t.Errorf("expected repo error, got %v", err)
	}
	if _, err := OpenURL("https://gitee.com/api/v5", "/", "repo"); err == nil {
		t.Errorf("expected error for slash-only owner, got nil")
	}
}

// TestOpenURLInvalidBase asserts a malformed base URL surfaces an error.
func TestOpenURLInvalidBase(t *testing.T) {
	if _, err := OpenURL("://bad", "owner", "repo"); err == nil ||
		!strings.Contains(err.Error(), "invalid platform base URL") {
		t.Errorf("expected invalid base error, got %v", err)
	}
}
