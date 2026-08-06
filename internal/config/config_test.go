package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToken_fromEnv(t *testing.T) {
	t.Setenv("GITEE_TOKEN", "env-token")
	tok, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	if tok != "env-token" {
		t.Errorf("expected env-token, got %s", tok)
	}
}

func TestToken_notLoggedIn(t *testing.T) {
	t.Setenv("GITEE_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	_, err := Token()
	if err == nil {
		t.Error("expected error when no token set")
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	t.Setenv("GITEE_TOKEN", "")

	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := SaveToken("my-saved-token"); err != nil {
		t.Fatal(err)
	}
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	tok, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	if tok != "my-saved-token" {
		t.Errorf("expected my-saved-token, got %s", tok)
	}
}

func TestAPIPrefix_fromEnv(t *testing.T) {
	t.Setenv("GITEE_API_PREFIX", "https://custom.example.com/api/v5")
	got := APIPrefix()
	if got != "https://custom.example.com/api/v5" {
		t.Errorf("expected custom prefix, got %s", got)
	}
}

func TestAPIPrefix_default(t *testing.T) {
	t.Setenv("GITEE_API_PREFIX", "")
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	got := APIPrefix()
	if got != DefaultAPIPrefix {
		t.Errorf("expected %s, got %s", DefaultAPIPrefix, got)
	}
}

func TestUpdateCheckCanBeDisabled(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if !UpdateCheckEnabled() {
		t.Fatal("update checks should be enabled by default")
	}
	if err := Set(KeyUpdateCheck, "false"); err != nil {
		t.Fatal(err)
	}
	if UpdateCheckEnabled() {
		t.Fatal("update checks should be disabled after setting update_check=false")
	}
}

func TestAICredentialPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		giteeToken string
		openAIKey  string
		want       string
	}{
		{name: "Gitee environment overrides OpenAI and stored token", giteeToken: "gitee-env", openAIKey: "openai-env", want: "gitee-env"},
		{name: "OpenAI environment overrides stored token", openAIKey: "openai-env", want: "openai-env"},
		{name: "stored token is the fallback", want: "stored-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
			t.Setenv("GITEE_AI_TOKEN", tt.giteeToken)
			t.Setenv("OPENAI_API_KEY", tt.openAIKey)
			if err := Load(); err != nil {
				t.Fatal(err)
			}
			if err := Set(KeyAIBaseURL, "https://api.openai.com/v1"); err != nil {
				t.Fatal(err)
			}
			if err := Set(KeyAIToken, "stored-token"); err != nil {
				t.Fatal(err)
			}

			cfg, err := AI()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Token != tt.want {
				t.Fatalf("expected token %q, got %q", tt.want, cfg.Token)
			}
		})
	}
}

func TestOpenAIAPIKeyIsRestrictedToOfficialEndpoint(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "openai-env")
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "official endpoint", baseURL: "https://api.openai.com/v1", want: "openai-env"},
		{name: "official endpoint case insensitive", baseURL: "https://API.OPENAI.COM/v1", want: "openai-env"},
		{name: "insecure official endpoint", baseURL: "http://api.openai.com/v1"},
		{name: "lookalike domain", baseURL: "https://api.openai.com.example.com/v1"},
		{name: "URL with credentials", baseURL: "https://user@api.openai.com/v1"},
		{name: "compatible third party", baseURL: "https://api.deepseek.com/v1"},
		{name: "invalid URL", baseURL: "://api.openai.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := openAIAPIKey(tt.baseURL); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestConfigDir_fromEnv(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", "/tmp/test-gitee-config")
	got := ConfigDir()
	if got != "/tmp/test-gitee-config" {
		t.Errorf("expected /tmp/test-gitee-config, got %s", got)
	}
}

func TestConfigDir_default(t *testing.T) {
	t.Setenv("GITEE_CONFIG_DIR", "")
	got := ConfigDir()
	home, _ := os.UserHomeDir()
	want := home + "/.config/gitee"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := Set("editor", "vim"); err != nil {
		t.Fatal(err)
	}
	got := Get("editor")
	if got != "vim" {
		t.Errorf("expected vim, got %s", got)
	}
}

func TestDeleteToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	t.Setenv("GITEE_TOKEN", "")

	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := SaveToken("deleteme"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteToken(); err != nil {
		t.Fatal(err)
	}
	_, err := os.Stat(credentialsPath())
	if !os.IsNotExist(err) {
		t.Error("expected credentials file to be removed after DeleteToken")
	}
}

func TestSensitiveSettingsUseCredentialsFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}

	if err := SaveToken("gitee-secret"); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyAIToken, "ai-secret"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(credentialsPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("expected credentials mode 0600, got %o", got)
	}
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	if !strings.Contains(contents, "gitee-secret") || !strings.Contains(contents, "ai-secret") {
		t.Fatalf("credentials file did not preserve both tokens: %s", contents)
	}
	if _, err := os.Stat(configPath()); !os.IsNotExist(err) {
		t.Fatalf("sensitive settings should not create config.yml, got error %v", err)
	}
}

func TestCredentialsArePreservedAcrossUpdatesAndLogout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyAIToken, "ai-secret"); err != nil {
		t.Fatal(err)
	}
	if err := SaveToken("gitee-secret"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteToken(); err != nil {
		t.Fatal(err)
	}

	credentials, err := readCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Token != "" {
		t.Errorf("expected default token to be deleted, got %q", credentials.Token)
	}
	if credentials.AI.Token != "ai-secret" {
		t.Fatalf("logout removed unrelated credentials: %+v", credentials)
	}
}

func TestGeneralConfigNeverContainsCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := SaveToken("gitee-secret"); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyAIToken, "ai-secret"); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyEditor, "vim"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath())
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	for _, secret := range []string{"gitee-secret", "ai-secret"} {
		if strings.Contains(contents, secret) {
			t.Errorf("config.yml contains secret %q: %s", secret, contents)
		}
	}
}

func TestConfigWriteMigratesLegacyCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	legacy := []byte("editor: nano\ntoken: old-gitee\nai:\n  token: old-ai\n  model: custom-model\n  base_url: https://ai.example.com/v1\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), legacy, 0644); err != nil {
		t.Fatal(err)
	}

	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyEditor, "vim"); err != nil {
		t.Fatal(err)
	}
	if got, _ := Token(); got != "old-gitee" {
		t.Errorf("expected migrated Gitee token, got %q", got)
	}
	aiConfig, err := AI()
	if err != nil {
		t.Fatal(err)
	}
	if aiConfig.Token != "old-ai" || aiConfig.Model != "custom-model" {
		t.Errorf("unexpected migrated AI config: %+v", aiConfig)
	}

	configData, err := os.ReadFile(configPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"old-gitee", "old-ai"} {
		if strings.Contains(string(configData), secret) {
			t.Errorf("legacy secret %q remains in config.yml", secret)
		}
	}
	credentialsInfo, err := os.Stat(credentialsPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := credentialsInfo.Mode().Perm(); got != 0600 {
		t.Errorf("expected migrated credentials mode 0600, got %o", got)
	}
}

func TestLoadDoesNotCreateConfigDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "missing")
	t.Setenv("GITEE_CONFIG_DIR", dir)

	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("Load created config directory or returned unexpected stat error: %v", err)
	}
}

func TestLoadDoesNotChangeCredentialPermissions(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Unix file modes are not supported on Windows")
	}
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	path := filepath.Join(dir, "credentials.yml")
	if err := os.WriteFile(path, []byte("token: read-only-token\n"), 0400); err != nil {
		t.Fatal(err)
	}

	if err := Load(); err != nil {
		t.Fatal(err)
	}
	token, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	if token != "read-only-token" {
		t.Fatalf("unexpected token %q", token)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0400 {
		t.Fatalf("Load changed credential permissions to %o", got)
	}
}

func TestSettingsAreRedacted(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEE_CONFIG_DIR", dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if err := SaveToken("gitee-secret"); err != nil {
		t.Fatal(err)
	}
	if err := Set(KeyAIToken, "ai-secret"); err != nil {
		t.Fatal(err)
	}

	if got := DisplayValue(KeyAIToken); got != RedactedValue {
		t.Errorf("expected redacted display value, got %q", got)
	}
	settings := AllSettings()
	serialized := fmt.Sprintf("%v", settings)
	for _, secret := range []string{"gitee-secret", "ai-secret"} {
		if strings.Contains(serialized, secret) {
			t.Errorf("settings expose secret %q: %s", secret, serialized)
		}
	}
}
