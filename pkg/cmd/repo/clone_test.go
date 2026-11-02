package repo

import (
	"encoding/base64"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func TestResolveCloneTarget(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		useSSH        bool
		setup         func(t *testing.T)
		wantURL       string
		wantErr       bool
		wantAuthHost  string
		wantAuthToken string
	}{
		{
			name:          "https URL authenticates for gitee.com",
			input:         "https://gitee.com/xxx/private_repo",
			wantURL:       "https://gitee.com/xxx/private_repo",
			wantAuthHost:  "gitee.com",
			wantAuthToken: "test-token",
		},
		{
			name:          "https URL with .git authenticates for gitee.com",
			input:         "https://gitee.com/xxx/private_repo.git",
			wantURL:       "https://gitee.com/xxx/private_repo.git",
			wantAuthHost:  "gitee.com",
			wantAuthToken: "test-token",
		},
		{
			name:    "scp style SSH passes through without auth",
			input:   "git@gitee.com:xxx/private_repo.git",
			wantURL: "git@gitee.com:xxx/private_repo.git",
		},
		{
			name:    "unknown host passes through without auth",
			input:   "https://github.com/owner/repo",
			wantURL: "https://github.com/owner/repo",
		},
		{
			name:    "port URL passes through without auth",
			input:   "https://gitee.com:8443/xxx/private_repo",
			wantURL: "https://gitee.com:8443/xxx/private_repo",
		},
		{
			name:  "http URL falls through to git credential handling",
			input: "http://gitee.com/owner/repo",
			setup: func(t *testing.T) {
				t.Setenv("GITEE_TOKEN", "test-token")
			},
			wantURL: "http://gitee.com/owner/repo",
		},
		{
			name:          "owner/repo expands to https",
			input:         "owner/repo",
			wantURL:       "https://gitee.com/owner/repo.git",
			wantAuthHost:  "gitee.com",
			wantAuthToken: "test-token",
		},
		{
			name:    "owner/repo with ssh expands to ssh",
			input:   "owner/repo",
			useSSH:  true,
			wantURL: "git@gitee.com:owner/repo.git",
		},
		{
			name:  "configured private host authenticates",
			input: "https://git.example.com/group/repo",
			setup: func(t *testing.T) {
				if err := config.SaveHostConfig("git.example.com", "host-token", ""); err != nil {
					t.Fatalf("SaveHostConfig: %v", err)
				}
			},
			wantURL:       "https://git.example.com/group/repo",
			wantAuthHost:  "git.example.com",
			wantAuthToken: "host-token",
		},
		{
			name:    "invalid input fails",
			input:   "not-a-repo",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GITEE_CONFIG_DIR", t.TempDir())
			t.Setenv("GIT_CONFIG_COUNT", "")
			t.Setenv("GITEE_TOKEN", "")
			if tc.setup != nil {
				tc.setup(t)
			}
			if tc.wantAuthHost == "gitee.com" {
				t.Setenv("GITEE_TOKEN", "test-token")
			}

			cloneURL, cloneEnv, err := resolveCloneTarget(&cmdutil.Factory{Hostname: "gitee.com"}, tc.input, tc.useSSH)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cloneURL != tc.wantURL {
				t.Errorf("cloneURL: want %q, got %q", tc.wantURL, cloneURL)
			}
			if tc.wantAuthHost == "" {
				if cloneEnv != nil {
					t.Errorf("cloneEnv: want nil, got %v", cloneEnv)
				}
				return
			}
			wantHeader := "GIT_CONFIG_VALUE_"
			wantKey := "http.https://" + tc.wantAuthHost + "/.extraHeader"
			wantValue := base64.StdEncoding.EncodeToString([]byte("oauth2:" + tc.wantAuthToken))
			gotKey := false
			gotValue := false
			gotPrompt := false
			for _, item := range cloneEnv {
				if strings.HasPrefix(item, "GIT_CONFIG_KEY_") && strings.HasSuffix(item, "="+wantKey) {
					gotKey = true
				}
				if strings.HasPrefix(item, wantHeader) && strings.Contains(item, wantValue) {
					gotValue = true
				}
				if item == "GIT_TERMINAL_PROMPT=0" {
					gotPrompt = true
				}
			}
			if !gotKey {
				t.Errorf("missing git config key for host %q in %v", tc.wantAuthHost, cloneEnv)
			}
			if !gotValue {
				t.Errorf("missing auth header value in %v", cloneEnv)
			}
			if !gotPrompt {
				t.Errorf("missing GIT_TERMINAL_PROMPT=0 in %v", cloneEnv)
			}
		})
	}
}

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
