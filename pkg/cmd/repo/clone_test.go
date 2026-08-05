package repo

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGitHTTPSAuthEnvKeepsTokenOutOfConfigKey(t *testing.T) {
	t.Setenv("GIT_CONFIG_COUNT", "2")
	token := "secret-token"
	env := gitHTTPSAuthEnv("gitee.example.com", token)

	wantHeader := "GIT_CONFIG_VALUE_2=Authorization: Basic " +
		base64.StdEncoding.EncodeToString([]byte("oauth2:"+token))
	want := map[string]bool{
		"GIT_CONFIG_KEY_2=http.https://gitee.example.com/.extraHeader": false,
		wantHeader:              false,
		"GIT_CONFIG_COUNT=3":    false,
		"GIT_TERMINAL_PROMPT=0": false,
	}
	for _, item := range env {
		if _, ok := want[item]; ok {
			want[item] = true
		}
		if strings.HasPrefix(item, "GIT_CONFIG_KEY_") && strings.Contains(item, token) {
			t.Fatalf("token leaked into Git config key: %q", item)
		}
	}
	for item, found := range want {
		if !found {
			t.Errorf("missing environment entry %q", item)
		}
	}
}
