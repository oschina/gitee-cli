package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitee.com/oschina/gitee-cli/internal/build"
	"golang.org/x/mod/semver"
)

type ReleaseInfo struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

type cacheEntry struct {
	CheckedAt time.Time    `json:"checked_at"`
	Latest    *ReleaseInfo `json:"latest,omitempty"`
}

var httpGetFn = func(url string) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := newUpdateRequest(url)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func newUpdateRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", build.UserAgent())
	return req, nil
}

// ensureVPrefix adds "v" prefix — semver.Compare requires it.
func ensureVPrefix(v string) string {
	if v == "" {
		return v
	}
	if v[0] != 'v' {
		return "v" + v
	}
	return v
}

func CheckForUpdate(currentVersion string) (*ReleaseInfo, error) {
	if currentVersion == "" || strings.HasSuffix(currentVersion, "-dev") {
		return nil, nil
	}

	normalized := ensureVPrefix(currentVersion)
	if !semver.IsValid(normalized) {
		return nil, nil
	}

	resp, err := httpGetFn("https://gitee.com/api/v5/repos/oschina/gitee-cli/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	latestVersion := ensureVPrefix(release.TagName)
	if !semver.IsValid(latestVersion) {
		return nil, nil
	}

	if semver.Compare(latestVersion, normalized) <= 0 {
		return nil, nil
	}

	return &ReleaseInfo{Version: release.TagName, URL: release.HTMLURL}, nil
}

func CheckForUpdateCached(currentVersion, cachePath string, maxAge time.Duration) (*ReleaseInfo, error) {
	if currentVersion == "" || strings.HasSuffix(currentVersion, "-dev") {
		return nil, nil
	}
	normalized := ensureVPrefix(currentVersion)
	if !semver.IsValid(normalized) {
		return nil, nil
	}

	if cached, ok := readFreshCache(cachePath, maxAge); ok {
		return cached.Latest, nil
	}

	entry := cacheEntry{CheckedAt: time.Now().UTC()}
	_ = writeCache(cachePath, entry)
	info, err := CheckForUpdate(currentVersion)
	if err != nil {
		return nil, err
	}
	entry.Latest = info
	_ = writeCache(cachePath, entry)
	return info, nil
}

func ShouldCheck(configEnabled, quiet bool) bool {
	if !configEnabled || quiet || envEnabled("GITEE_NO_UPDATE_NOTIFIER") {
		return false
	}
	for _, name := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "GITEE_CI"} {
		if envEnabled(name) {
			return false
		}
	}
	return true
}

func envEnabled(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value != "" && value != "0" && value != "false" && value != "no"
}

func readFreshCache(path string, maxAge time.Duration) (cacheEntry, bool) {
	var entry cacheEntry
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &entry) != nil || entry.CheckedAt.IsZero() {
		return entry, false
	}
	return entry, time.Since(entry.CheckedAt) >= 0 && time.Since(entry.CheckedAt) < maxAge
}

func writeCache(path string, entry cacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".update-check-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
