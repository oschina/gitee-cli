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

func TestGiteeRemotes_parseOnly(t *testing.T) {
	cases := []struct {
		url   string
		isGit bool
	}{
		{"https://gitee.com/foo/bar.git", true},
		{"git@gitee.com:foo/bar.git", true},
		{"https://git.company.com/foo/bar.git", true},
		{"git@git.company.com:foo/bar.git", true},
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
