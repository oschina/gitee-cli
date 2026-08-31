package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndGetHostConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	if err := SaveHostConfig("git.example.com", "tok123", "https://git.example.com/api/v5"); err != nil {
		t.Fatal(err)
	}

	hc, ok := GetHostConfig("git.example.com")
	if !ok {
		t.Fatal("expected host config to exist")
	}
	if hc.Token != "tok123" {
		t.Errorf("expected tok123, got %s", hc.Token)
	}
	if hc.APIPrefix != "https://git.example.com/api/v5" {
		t.Errorf("unexpected api_prefix: %s", hc.APIPrefix)
	}
}

func TestGetHostConfig_notFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	_, ok := GetHostConfig("nonexistent.com")
	if ok {
		t.Error("expected not found")
	}
}

func TestListHosts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	if err := SaveHostConfig("host1.com", "t1", ""); err != nil {
		t.Fatal(err)
	}
	if err := SaveHostConfig("host2.com", "t2", ""); err != nil {
		t.Fatal(err)
	}

	hosts := ListHosts()
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d: %v", len(hosts), hosts)
	}
}

func TestDefaultHostname_usesOnlyPrivateHostWithoutDefaultCredentials(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_TOKEN", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := SaveHostConfig("git.example.com", "private-token", ""); err != nil {
		t.Fatal(err)
	}

	if got := DefaultHostname(); got != "git.example.com" {
		t.Fatalf("expected only private host, got %q", got)
	}
}

func TestDefaultHostname_prefersDefaultHostWithCredentials(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_TOKEN", "default-token")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := SaveHostConfig("git.example.com", "private-token", ""); err != nil {
		t.Fatal(err)
	}

	if got := DefaultHostname(); got != DefaultHost {
		t.Fatalf("expected %q, got %q", DefaultHost, got)
	}
}

func TestDefaultHostname_prefersConfiguredPrivateHost(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_TOKEN", "default-token")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyHost, "git.example.com"); err != nil {
		t.Fatal(err)
	}

	if got := DefaultHostname(); got != "git.example.com" {
		t.Fatalf("expected configured private host, got %q", got)
	}
}

func TestDeleteHostConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	if err := SaveHostConfig("delete-me.com", "tok", ""); err != nil {
		t.Fatal(err)
	}
	if err := DeleteHostConfig("delete-me.com"); err != nil {
		t.Fatal(err)
	}

	v := hostsViper()
	if v.IsSet("delete-me.com") {
		t.Error("expected host to be absent from hosts.yml")
	}
}

func TestDeleteHostConfig_notFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	err := DeleteHostConfig("ghost.com")
	if err == nil {
		t.Error("expected error for nonexistent host")
	}
}

func TestTokenForHost_default(t *testing.T) {
	t.Setenv("GITEE_TOKEN", "default-token")
	tok, err := TokenForHost("")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "default-token" {
		t.Errorf("expected default-token, got %s", tok)
	}
}

func TestTokenForHost_custom(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	t.Setenv("GITEE_TOKEN", "")

	if err := SaveHostConfig("custom.com", "custom-tok", ""); err != nil {
		t.Fatal(err)
	}

	tok, err := TokenForHost("custom.com")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "custom-tok" {
		t.Errorf("expected custom-tok, got %s", tok)
	}
}

func TestTokenForHost_notLoggedIn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	_, err := TokenForHost("noauth.com")
	if err == nil {
		t.Error("expected error for unauthenticated host")
	}
}

func TestAPIPrefixForHost_default(t *testing.T) {
	t.Setenv("GITEE_API_PREFIX", "")
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := APIPrefixForHost("")
	if got != DefaultAPIPrefix {
		t.Errorf("expected %s, got %s", DefaultAPIPrefix, got)
	}
}

func TestAPIPrefixForHost_custom(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	if err := SaveHostConfig("git.co", "tok", "https://git.co/api/v5"); err != nil {
		t.Fatal(err)
	}

	got := APIPrefixForHost("git.co")
	if got != "https://git.co/api/v5" {
		t.Errorf("unexpected api prefix: %s", got)
	}
}

func TestAPIPrefixForHost_fallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	got := APIPrefixForHost("unknown.io")
	want := "https://unknown.io/api/v5"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestHostsFilePath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)

	if err := SaveHostConfig("a.com", "tok", ""); err != nil {
		t.Fatal(err)
	}

	hostsFile := filepath.Join(dir, "hosts.yml")
	if _, err := os.Stat(hostsFile); os.IsNotExist(err) {
		t.Error("expected hosts.yml to exist")
	}
	info, err := os.Stat(hostsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("expected hosts.yml mode 0600, got %o", got)
	}
}

func TestGoAPIBaseURL_noRepo(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("gitee.com", "")
	want := "https://go-api.gitee.com" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIBaseURL_withRepo(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("gitee.com", "acme/gitee-go-plugin")
	want := "https://go-api.gitee.com/acme/gitee-go-plugin" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIBaseURL_sameHostFallback(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("git.company.com", "team/plugin")
	want := "https://git.company.com/go-api/team/plugin" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

// premium/private deployment: host stays the Gitee host, /go-api fixed segment.
func TestGoAPIBaseURL_premiumDeployment(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("premium-k8s.gitee.cn", "autodeploy/gitee-go-pipeline")
	want := "https://premium-k8s.gitee.cn/go-api/autodeploy/gitee-go-pipeline" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIBaseURL_explicitHostOverride(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyGoAPIHost, "go.custom.example"); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("git.company.com", "")
	want := "https://go.custom.example" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIHost_runjsHost(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIHost("xxx.runjs.cn")
	want := LocalGoAPIHost
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIHost_runjsHost_otherSubdomain(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIHost("my-org.runjs.cn")
	want := LocalGoAPIHost
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIBaseURL_runjsHost_noRepo(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("xxx.runjs.cn", "")
	want := "https://local-pipe-api.runjs.cn" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIBaseURL_runjsHost_withRepo(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("xxx.runjs.cn", "acme/gitee-go-plugin")
	want := "https://local-pipe-api.runjs.cn/acme/gitee-go-plugin" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestGoAPIBaseURL_runjsHost_explicitOverride(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	t.Setenv("GITEE_GO_API_HOST", "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyGoAPIHost, "go.custom.example"); err != nil {
		t.Fatal(err)
	}
	got := GoAPIBaseURL("xxx.runjs.cn", "team/plugin")
	want := "https://go.custom.example/team/plugin" + GoAPIBasePath
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}
