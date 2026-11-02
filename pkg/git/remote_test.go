package git

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https URL",
			url:       "https://gitee.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "https URL without .git",
			url:       "https://gitee.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH URL",
			url:       "git@gitee.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH URL without .git",
			url:       "git@gitee.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "private deployment https",
			url:       "https://git.company.com/team/project.git",
			wantOwner: "team",
			wantRepo:  "project",
		},
		{
			name:      "private deployment SSH",
			url:       "git@git.company.com:team/project.git",
			wantOwner: "team",
			wantRepo:  "project",
		},
		{
			name:      "https URL with userinfo",
			url:       "https://alice@gitee.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "https URL with userinfo and password",
			url:       "https://alice:secret@gitee.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "enterprise three-segment https URL",
			url:       "https://gitee.com/ent-a/group-b/repo-c.git",
			wantOwner: "ent-a/group-b",
			wantRepo:  "repo-c",
		},
		{
			name:      "enterprise three-segment SSH URL",
			url:       "git@gitee.com:ent-a/group-b/repo-c.git",
			wantOwner: "ent-a/group-b",
			wantRepo:  "repo-c",
		},
		{
			name:      "enterprise three-segment https URL with userinfo",
			url:       "https://alice@gitee.com/ent-a/group-b/repo-c.git",
			wantOwner: "ent-a/group-b",
			wantRepo:  "repo-c",
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "local path",
			url:     "/path/to/repo",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, err := parseRemoteURL(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (owner=%s repo=%s)", owner, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tc.wantOwner {
				t.Errorf("owner: want %q, got %q", tc.wantOwner, owner)
			}
			if repo != tc.wantRepo {
				t.Errorf("repo: want %q, got %q", tc.wantRepo, repo)
			}
		})
	}
}

func TestHTTPRemoteHost(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		wantHost string
		wantOK   bool
	}{
		{
			name:     "https basic",
			url:      "https://gitee.com/owner/repo",
			wantHost: "gitee.com",
			wantOK:   true,
		},
		{
			name:     "https with .git",
			url:      "https://gitee.com/owner/repo.git",
			wantHost: "gitee.com",
			wantOK:   true,
		},
		{
			name:     "https with userinfo",
			url:      "https://user@gitee.com/owner/repo",
			wantHost: "gitee.com",
			wantOK:   true,
		},
		{
			name:     "http",
			url:      "http://gitee.com/owner/repo",
			wantHost: "gitee.com",
			wantOK:   true,
		},
		{
			name:     "uppercase scheme",
			url:      "HTTPS://gitee.com/owner/repo",
			wantHost: "gitee.com",
			wantOK:   true,
		},
		{
			name:     "private host",
			url:      "https://git.example.com/group/repo",
			wantHost: "git.example.com",
			wantOK:   true,
		},
		{
			name: "scp style SSH",
			url:  "git@gitee.com:owner/repo.git",
		},
		{
			name: "ssh scheme",
			url:  "ssh://git@gitee.com/owner/repo.git",
		},
		{
			name: "git scheme",
			url:  "git://gitee.com/owner/repo.git",
		},
		{
			name: "file scheme",
			url:  "file:///path/to/repo",
		},
		{
			name: "port URL",
			url:  "https://gitee.com:8443/owner/repo",
		},
		{
			name: "no path",
			url:  "https://gitee.com",
		},
		{
			name: "empty string",
			url:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, ok := HTTPRemoteHost(tc.url)
			if ok != tc.wantOK {
				t.Fatalf("ok: want %v, got %v", tc.wantOK, ok)
			}
			if host != tc.wantHost {
				t.Errorf("host: want %q, got %q", tc.wantHost, host)
			}
		})
	}
}

func TestGiteeRemotes_parseOnly(t *testing.T) {
	cases := []struct {
		url   string
		isGit bool
	}{
		{"https://gitee.com/foo/bar.git", true},
		{"git@gitee.com:foo/bar.git", true},
		{"https://git.company.com/foo/bar.git", true},
		{"git@git.company.com:foo/bar.git", true},
		{"https://alice@gitee.com/foo/bar.git", true},
		{"https://gitee.com/ent/group/repo.git", true},
		{"git@gitee.com:ent/group/repo.git", true},
		{"https://github.com/foo/bar.git", true},
		{"https://gitlab.com/foo/bar.git", true},
		{"", false},
		{"local-path", false},
	}
	for _, tc := range cases {
		m := remoteURLPattern.MatchString(tc.url)
		if m != tc.isGit {
			t.Errorf("url %q: expected isGit=%v, got %v", tc.url, tc.isGit, m)
		}
	}
}
