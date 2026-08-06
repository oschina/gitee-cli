package update

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gitee.com/oschina/gitee-cli/internal/build"
)

func TestNewUpdateRequestUserAgent(t *testing.T) {
	req, err := newUpdateRequest("https://gitee.com/api/v5/releases/latest")
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("User-Agent"); got != build.UserAgent() {
		t.Fatalf("expected %q user agent, got %q", build.UserAgent(), got)
	}
}

func TestCheckForUpdate_devVersion(t *testing.T) {
	info, err := CheckForUpdate("0.1.0-dev")
	if err != nil {
		t.Fatal(err)
	}
	if info != nil {
		t.Error("expected nil for dev version")
	}
}

func TestVersionClassification(t *testing.T) {
	if !IsReleaseVersion("1.2.3") {
		t.Fatal("expected valid release versions")
	}
	if IsReleaseVersion("dev+abc123") {
		t.Fatal("unexpected version classification")
	}
}

func TestCheckForUpdate_emptyVersion(t *testing.T) {
	info, err := CheckForUpdate("")
	if err != nil {
		t.Fatal(err)
	}
	if info != nil {
		t.Error("expected nil for empty version")
	}
}

func TestCheckForUpdate_sameVersion(t *testing.T) {
	srv := newReleasesServer(t, "v1.0.0", "https://example.com/v1.0.0")
	defer srv.Close()

	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) { return http.Get(srv.URL) }

	info, err := CheckForUpdate("v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if info != nil {
		t.Error("expected nil when current version matches latest")
	}
}

func TestCheckForUpdate_newVersion(t *testing.T) {
	srv := newReleasesServer(t, "v2.0.0", "https://example.com/v2.0.0")
	defer srv.Close()

	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) { return http.Get(srv.URL) }

	info, err := CheckForUpdate("v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil ReleaseInfo for new version")
	}
	if info.Version != "v2.0.0" {
		t.Errorf("expected version v2.0.0, got %s", info.Version)
	}
	if info.URL != "https://gitee.com/oschina/gitee-cli/releases/tag/v2.0.0" {
		t.Errorf("unexpected release URL: %s", info.URL)
	}
}

func TestCheckForUpdate_serverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) { return http.Get(srv.URL) }

	_, err := CheckForUpdate("v1.0.0")
	if err == nil {
		t.Error("expected error on server error")
	}
}

func TestCheckForUpdate_semver_compare(t *testing.T) {
	// v0.10.0 > v0.2.0 — pure string comparison would miss this
	srv := newReleasesServer(t, "v0.10.0", "https://example.com/v0.10.0")
	defer srv.Close()

	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) { return http.Get(srv.URL) }

	info, err := CheckForUpdate("v0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil for v0.10.0 > v0.2.0 (semver)")
	}
	if info.Version != "v0.10.0" {
		t.Errorf("expected v0.10.0, got %s", info.Version)
	}
}

func TestCheckForUpdate_semver_noVPrefix(t *testing.T) {
	// Current version without "v" prefix should still work
	srv := newReleasesServer(t, "v3.0.0", "https://example.com/v3.0.0")
	defer srv.Close()

	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) { return http.Get(srv.URL) }

	info, err := CheckForUpdate("2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil for v3.0.0 > 2.0.0")
	}
}

func TestCheckForUpdate_currentNewer(t *testing.T) {
	// Current version is newer than latest release — no update needed
	srv := newReleasesServer(t, "v1.0.0", "https://example.com/v1.0.0")
	defer srv.Close()

	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) { return http.Get(srv.URL) }

	info, err := CheckForUpdate("v2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if info != nil {
		t.Error("expected nil when current is newer than latest")
	}
}

func TestCheckForUpdate_returnsLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releaseResponse{
			TagName: "v1.10.0",
			Assets:  []ReleaseAsset{{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"}},
		})
	}))
	defer srv.Close()
	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) { return http.Get(srv.URL) }

	info, err := CheckForUpdate("v0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Version != "v1.10.0" {
		t.Fatalf("unexpected release: %#v", info)
	}
	if info.URL != "https://gitee.com/oschina/gitee-cli/releases/tag/v1.10.0" {
		t.Fatalf("unexpected release URL: %s", info.URL)
	}
	if len(info.Assets) != 1 {
		t.Fatalf("expected assets to be preserved, got %#v", info.Assets)
	}
}

func TestCheckForUpdateCached_reusesFreshResult(t *testing.T) {
	srv := newReleasesServer(t, "v2.0.0", "https://example.com/v2.0.0")
	defer srv.Close()

	requests := 0
	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) {
		requests++
		return http.Get(srv.URL)
	}

	cachePath := filepath.Join(t.TempDir(), "update-check.json")
	for i := 0; i < 2; i++ {
		info, err := CheckForUpdateCached("v1.0.0", cachePath, 24*time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if info == nil || info.Version != "v2.0.0" {
			t.Fatalf("unexpected cached release: %+v", info)
		}
	}
	if requests != 1 {
		t.Fatalf("expected one network request, got %d", requests)
	}
}

func TestCheckForUpdateCached_invalidatesAfterCurrentVersionChanges(t *testing.T) {
	srv := newReleasesServer(t, "v2.0.0", "")
	defer srv.Close()
	requests := 0
	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) {
		requests++
		return http.Get(srv.URL)
	}
	cachePath := filepath.Join(t.TempDir(), "update-check.json")
	if info, err := CheckForUpdateCached("v1.0.0", cachePath, 24*time.Hour); err != nil || info == nil {
		t.Fatalf("first check = %#v, %v", info, err)
	}
	if info, err := CheckForUpdateCached("v2.0.0", cachePath, 24*time.Hour); err != nil || info != nil {
		t.Fatalf("post-upgrade check = %#v, %v", info, err)
	}
	if requests != 2 {
		t.Fatalf("expected cache invalidation to make two requests, got %d", requests)
	}
}

func TestCheckForUpdateCached_doesNotCacheFailure(t *testing.T) {
	srv := newReleasesServer(t, "v2.0.0", "")
	defer srv.Close()
	requests := 0
	origFn := httpGetFn
	defer func() { httpGetFn = origFn }()
	httpGetFn = func(url string) (*http.Response, error) {
		requests++
		if requests == 1 {
			return nil, errors.New("temporary network failure")
		}
		return http.Get(srv.URL)
	}
	cachePath := filepath.Join(t.TempDir(), "update-check.json")
	if _, err := CheckForUpdateCached("v1.0.0", cachePath, 24*time.Hour); err == nil {
		t.Fatal("expected first check to fail")
	}
	if info, err := CheckForUpdateCached("v1.0.0", cachePath, 24*time.Hour); err != nil || info == nil {
		t.Fatalf("retry = %#v, %v", info, err)
	}
	if requests != 2 {
		t.Fatalf("expected a retry after failure, got %d requests", requests)
	}
}

func TestShouldCheck(t *testing.T) {
	for _, name := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "GITEE_CI", "GITEE_NO_UPDATE_NOTIFIER"} {
		t.Setenv(name, "")
	}
	if !ShouldCheck(true, false) {
		t.Fatal("expected update checks to be enabled")
	}
	if ShouldCheck(false, false) || ShouldCheck(true, true) {
		t.Fatal("disabled or quiet commands must not check for updates")
	}
	t.Setenv("CI", "true")
	if ShouldCheck(true, false) {
		t.Fatal("CI must disable update checks")
	}
	t.Setenv("CI", "false")
	t.Setenv("GITEE_NO_UPDATE_NOTIFIER", "1")
	if ShouldCheck(true, false) {
		t.Fatal("GITEE_NO_UPDATE_NOTIFIER must disable update checks")
	}
}

func newReleasesServer(t *testing.T, tagName, _ string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releaseResponse{TagName: tagName})
	}))
}
