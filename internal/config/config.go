package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

const (
	KeyToken         = "token"
	KeyHost          = "host"
	KeyEditor        = "editor"
	KeyPager         = "pager"
	KeyAPIPrefix     = "api_prefix"
	KeyAPISwaggerURL = "api_swagger_url"
	KeyTUI           = "tui"
	KeyColorize      = "colorize"
	KeyAIBaseURL     = "ai.base_url"
	KeyAIModel       = "ai.model"
	KeyAIToken       = "ai.token"
	KeyAILanguage    = "ai.language"
	KeyDefaultRepo   = "default_repo"
	KeyLocale        = "locale"
	KeyTheme         = "theme"
	KeyUpdateCheck   = "update_check"

	DefaultHost      = "gitee.com"
	DefaultAPIPrefix = "https://gitee.com/api/v5"
	DefaultAIModel   = "gpt-4o-mini"
	DefaultTheme     = "default"
	RedactedValue    = "<redacted>"
)

type credentialFile struct {
	Token string `yaml:"token,omitempty"`
	AI    struct {
		Token string `yaml:"token,omitempty"`
	} `yaml:"ai,omitempty"`
}

var ConfigOptions = map[string][]string{
	KeyTheme:       {"auto", "dark", "light", "dracula", "tokyo-night", "pink"},
	KeyLocale:      {"en", "zh_CN"},
	KeyTUI:         {"true", "false"},
	KeyColorize:    {"true", "false"},
	KeyUpdateCheck: {"true", "false"},
}

var ErrNotLoggedIn = errors.New("not logged in: run `gitee auth login` first")

func ConfigDir() string {
	if dir := os.Getenv("GITEE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gitee")
}

func credentialsPath() string {
	return filepath.Join(ConfigDir(), "credentials.yml")
}

func configPath() string {
	return filepath.Join(ConfigDir(), "config.yml")
}

func Load() error {
	viper.Reset()
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configPath())

	viper.SetDefault(KeyHost, DefaultHost)
	viper.SetDefault(KeyAPIPrefix, DefaultAPIPrefix)
	viper.SetDefault(KeyTUI, false)
	viper.SetDefault(KeyUpdateCheck, true)

	viper.SetEnvPrefix("GITEE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return fmt.Errorf("config: read: %w", err)
		}
	}

	return nil
}

func Get(key string) string {
	return viper.GetString(key)
}

func DefaultRepo() string {
	return viper.GetString(KeyDefaultRepo)
}

func Locale() string {
	if v := viper.GetString(KeyLocale); v != "" {
		return v
	}
	lang := os.Getenv("LANG")
	if strings.HasPrefix(strings.ToLower(lang), "zh") {
		return "zh_CN"
	}
	lcAll := os.Getenv("LC_ALL")
	if strings.HasPrefix(strings.ToLower(lcAll), "zh") {
		return "zh_CN"
	}
	return "en"
}

func Set(key, value string) error {
	normalizedKey := strings.ToLower(key)
	if IsSensitiveKey(normalizedKey) {
		if err := saveCredential(normalizedKey, value); err != nil {
			return err
		}
		viper.Set(normalizedKey, value)
		return nil
	}
	viper.Set(normalizedKey, value)
	return saveGeneralSettings()
}

func IsSensitiveKey(key string) bool {
	switch strings.ToLower(key) {
	case KeyToken, KeyAIToken:
		return true
	default:
		return false
	}
}

func Token() (string, error) {
	if t := os.Getenv("GITEE_TOKEN"); t != "" {
		return t, nil
	}
	t, err := credentialValue(KeyToken)
	if err != nil {
		return "", fmt.Errorf("read credentials: %w", err)
	}
	if t == "" {
		return "", ErrNotLoggedIn
	}
	return t, nil
}

func SaveToken(token string) error {
	if err := saveCredential(KeyToken, token); err != nil {
		return err
	}
	viper.Set(KeyToken, token)
	return nil
}

func DeleteToken() error {
	if err := migrateLegacyCredentials(); err != nil {
		return err
	}
	credentials, err := readCredentials()
	if err != nil {
		return err
	}
	credentials.Token = ""
	if err := writeCredentials(credentials); err != nil {
		return err
	}
	viper.Set(KeyToken, "")
	return nil
}

func APIPrefix() string {
	if p := os.Getenv("GITEE_API_PREFIX"); p != "" {
		return p
	}
	if p := viper.GetString(KeyAPIPrefix); p != "" {
		return p
	}
	return DefaultAPIPrefix
}

func APISwaggerURL() string {
	if u := viper.GetString(KeyAPISwaggerURL); u != "" {
		return u
	}
	return DefaultAPIPrefix + "/swagger_doc.json"
}

func AllSettings() map[string]interface{} {
	settings := viper.AllSettings()
	if credentials, err := readCredentials(); err == nil {
		if credentials.Token != "" {
			settings[KeyToken] = credentials.Token
		}
		if credentials.AI.Token != "" {
			aiSettings, _ := settings["ai"].(map[string]interface{})
			if aiSettings == nil {
				aiSettings = map[string]interface{}{}
				settings["ai"] = aiSettings
			}
			aiSettings["token"] = credentials.AI.Token
		}
	}
	redactSensitiveSettings(settings)
	return settings
}

func DisplayValue(key string) string {
	key = strings.ToLower(key)
	value := viper.GetString(key)
	if IsSensitiveKey(key) {
		if stored, err := credentialValue(key); err == nil {
			value = stored
		}
	}
	if value != "" && IsSensitiveKey(key) {
		return RedactedValue
	}
	return value
}

func readCredentials() (credentialFile, error) {
	var credentials credentialFile
	data, err := os.ReadFile(credentialsPath())
	if os.IsNotExist(err) {
		return credentials, nil
	}
	if err != nil {
		return credentials, err
	}
	if err := yaml.Unmarshal(data, &credentials); err != nil {
		return credentials, fmt.Errorf("decode credentials: %w", err)
	}
	return credentials, nil
}

func credentialValue(key string) (string, error) {
	credentials, err := readCredentials()
	if err != nil {
		return "", err
	}
	var value string
	switch key {
	case KeyToken:
		value = credentials.Token
	case KeyAIToken:
		value = credentials.AI.Token
	}
	if value != "" {
		return value, nil
	}
	return viper.GetString(key), nil
}

func saveCredential(key, value string) error {
	if err := migrateLegacyCredentials(); err != nil {
		return err
	}
	credentials, err := readCredentials()
	if err != nil {
		return err
	}
	switch key {
	case KeyToken:
		credentials.Token = value
	case KeyAIToken:
		credentials.AI.Token = value
	default:
		return fmt.Errorf("unsupported credential key %q", key)
	}
	return writeCredentials(credentials)
}

func writeCredentials(credentials credentialFile) error {
	if credentials.Token == "" && credentials.AI.Token == "" {
		err := os.Remove(credentialsPath())
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return writeYAMLAtomic(credentialsPath(), credentials, 0600)
}

func saveGeneralSettings() error {
	if err := migrateLegacyCredentials(); err != nil {
		return err
	}
	settings := viper.AllSettings()
	removeSensitiveSettings(settings)
	return writeYAMLAtomic(configPath(), settings, 0600)
}

func migrateLegacyCredentials() error {
	data, err := os.ReadFile(configPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	settings := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("decode legacy config: %w", err)
	}

	legacyToken, _ := settings[KeyToken].(string)
	legacyAIToken := ""
	if aiSettings, ok := settings["ai"].(map[string]interface{}); ok {
		legacyAIToken, _ = aiSettings["token"].(string)
	}
	if legacyToken == "" && legacyAIToken == "" {
		return nil
	}

	credentials, err := readCredentials()
	if err != nil {
		return err
	}
	if credentials.Token == "" {
		credentials.Token = legacyToken
	}
	if credentials.AI.Token == "" {
		credentials.AI.Token = legacyAIToken
	}
	if err := writeCredentials(credentials); err != nil {
		return err
	}

	removeSensitiveSettings(settings)
	return writeYAMLAtomic(configPath(), settings, 0600)
}

func removeSensitiveSettings(settings map[string]interface{}) {
	delete(settings, KeyToken)
	if aiSettings, ok := settings["ai"].(map[string]interface{}); ok {
		delete(aiSettings, "token")
		if len(aiSettings) == 0 {
			delete(settings, "ai")
		}
	}
}

func redactSensitiveSettings(settings map[string]interface{}) {
	redactSetting(settings, KeyToken)
	if aiSettings, ok := settings["ai"].(map[string]interface{}); ok {
		redactSetting(aiSettings, "token")
	}
}

func redactSetting(settings map[string]interface{}, key string) {
	value, exists := settings[key]
	if !exists || fmt.Sprintf("%v", value) == "" {
		delete(settings, key)
		return
	}
	settings[key] = RedactedValue
}

func writeYAMLAtomic(path string, value interface{}, perm os.FileMode) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return os.Chmod(path, perm)
}

func Reset() {
	viper.Reset()
}

func TUIEnabled() bool {
	return viper.GetBool(KeyTUI)
}

func ColorizeEnabled() bool {
	return viper.GetBool(KeyColorize)
}

func UpdateCheckEnabled() bool {
	return viper.GetBool(KeyUpdateCheck)
}

func UpdateCachePath() string {
	return filepath.Join(ConfigDir(), "update-check.json")
}

func PRTemplatePath() string {
	return filepath.Join(ConfigDir(), "pr_template.md")
}

// Editor returns the editor to use for interactive text input.
// Priority: $GIT_EDITOR > $VISUAL > $EDITOR > config key "editor" > "vi"
func Editor() string {
	for _, env := range []string{"GIT_EDITOR", "VISUAL", "EDITOR"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	if v := viper.GetString(KeyEditor); v != "" {
		return v
	}
	return "vim"
}

type AIConfig struct {
	BaseURL  string
	Model    string
	Token    string
	Language string
}

func AI() (AIConfig, error) {
	baseURL := viper.GetString(KeyAIBaseURL)
	token := os.Getenv("GITEE_AI_TOKEN")
	if token == "" {
		token = openAIAPIKey(baseURL)
	}
	if token == "" {
		var err error
		token, err = credentialValue(KeyAIToken)
		if err != nil {
			return AIConfig{}, fmt.Errorf("read AI credentials: %w", err)
		}
	}
	if token == "" {
		return AIConfig{}, fmt.Errorf("AI token not configured: run `gitee config set ai.token` or set GITEE_AI_TOKEN; OPENAI_API_KEY is accepted for api.openai.com")
	}
	if baseURL == "" {
		return AIConfig{}, fmt.Errorf("AI base URL not configured: run `gitee config set ai.base_url <url>`")
	}
	model := viper.GetString(KeyAIModel)
	if model == "" {
		model = DefaultAIModel
	}
	return AIConfig{
		BaseURL:  baseURL,
		Model:    model,
		Token:    token,
		Language: viper.GetString(KeyAILanguage),
	}, nil
}

func openAIAPIKey(baseURL string) string {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return ""
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "https" || u.User != nil || !strings.EqualFold(u.Hostname(), "api.openai.com") {
		return ""
	}
	return key
}
