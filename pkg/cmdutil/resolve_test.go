package cmdutil

import (
	"testing"
)

func TestParseOwnerRepo(t *testing.T) {
	cases := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"alice/my-project", "alice", "my-project", false},
		{"org/repo.git", "org", "repo.git", false},
		{"noslash", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
		{"", "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			owner, repo, err := ParseOwnerRepo(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got owner=%q repo=%q", owner, repo)
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
