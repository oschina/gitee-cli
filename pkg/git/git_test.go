package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initGitRepo(t *testing.T, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "remote", "add", "origin", remoteURL},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup cmd %v failed: %v\n%s", c, err, out)
		}
	}
	return dir
}

func TestParseRemoteURL_table(t *testing.T) {
	cases := []struct {
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"https://gitee.com/alice/myrepo.git", "alice", "myrepo", false},
		{"https://gitee.com/alice/myrepo", "alice", "myrepo", false},
		{"git@gitee.com:alice/myrepo.git", "alice", "myrepo", false},
		{"git@gitee.com:alice/myrepo", "alice", "myrepo", false},
		{"https://git.company.com/team/project.git", "team", "project", false},
		{"git@git.company.com:team/project.git", "team", "project", false},
		{"https://github.com/alice/myrepo.git", "alice", "myrepo", false},
		{"", "", "", true},
		{"not-a-url", "", "", true},
	}
	for _, tc := range cases {
		owner, repo, err := parseRemoteURL(tc.url)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseRemoteURL(%q): expected error, got nil (owner=%s repo=%s)", tc.url, owner, repo)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRemoteURL(%q): unexpected error: %v", tc.url, err)
			continue
		}
		if owner != tc.wantOwner || repo != tc.wantRepo {
			t.Errorf("parseRemoteURL(%q): got %s/%s, want %s/%s", tc.url, owner, repo, tc.wantOwner, tc.wantRepo)
		}
	}
}

func TestRemoteURLPattern(t *testing.T) {
	matched := []string{
		"https://gitee.com/foo/bar.git",
		"https://gitee.com/foo/bar",
		"git@gitee.com:foo/bar.git",
		"git@gitee.com:foo/bar",
		"https://git.company.com/foo/bar.git",
		"git@git.company.com:foo/bar.git",
		"https://github.com/foo/bar.git",
		"https://gitlab.com/foo/bar.git",
	}
	notMatched := []string{
		"",
		"local-path",
	}
	for _, u := range matched {
		if !remoteURLPattern.MatchString(u) {
			t.Errorf("expected %q to match remoteURLPattern", u)
		}
	}
	for _, u := range notMatched {
		if remoteURLPattern.MatchString(u) {
			t.Errorf("expected %q NOT to match remoteURLPattern", u)
		}
	}
}

func TestGiteeRemotes(t *testing.T) {
	dir := initGitRepo(t, "https://gitee.com/alice/testrepo.git")
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	remotes, err := GiteeRemotes()
	if err != nil {
		t.Fatal(err)
	}
	if len(remotes) == 0 {
		t.Fatal("expected at least one gitee remote")
	}
	if remotes[0].Name != "origin" {
		t.Errorf("expected remote name 'origin', got %q", remotes[0].Name)
	}
}

func TestGiteeRemotes_noGiteeRemote(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, err := GiteeRemotes()
	if err == nil {
		t.Error("expected error when no git remote")
	}
}

func TestRepoFromRemote(t *testing.T) {
	dir := initGitRepo(t, "https://gitee.com/testowner/testrepo.git")
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	owner, repo, err := RepoFromRemote()
	if err != nil {
		t.Fatal(err)
	}
	if owner != "testowner" {
		t.Errorf("expected owner 'testowner', got %q", owner)
	}
	if repo != "testrepo" {
		t.Errorf("expected repo 'testrepo', got %q", repo)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %v\n%s", c, err, out)
		}
	}

	f := filepath.Join(dir, "init.txt")
	os.WriteFile(f, []byte("init"), 0644)
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = dir
	addCmd.Run()
	commitCmd := exec.Command("git", "commit", "-m", "init")
	commitCmd.Dir = dir
	commitCmd.Run()

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	branch, err := CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestLocalBranches(t *testing.T) {
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		cmd.Run()
	}

	f := filepath.Join(dir, "init.txt")
	os.WriteFile(f, []byte("init"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()
	exec.Command("git", "-C", dir, "checkout", "-b", "feature/test").Run()

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	branches, err := LocalBranches()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range branches {
		if strings.Contains(b, "feature/test") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'feature/test' in branches, got: %v", branches)
	}
}

func TestBranchExists(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "T").Run()

	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("x"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()
	exec.Command("git", "-C", dir, "checkout", "-b", "mybranch").Run()

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	exists, err := BranchExists("mybranch")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected 'mybranch' to exist")
	}

	notExists, err := BranchExists("nonexistent-branch")
	if err != nil {
		t.Fatal(err)
	}
	if notExists {
		t.Error("expected 'nonexistent-branch' to not exist")
	}
}

func TestCheckout(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "T").Run()

	f := filepath.Join(dir, "f.txt")
	os.WriteFile(f, []byte("x"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()
	exec.Command("git", "-C", dir, "checkout", "-b", "target-branch").Run()
	exec.Command("git", "-C", dir, "checkout", "-").Run()

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	if err := Checkout("target-branch"); err != nil {
		t.Fatal(err)
	}

	out, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	got := strings.TrimSpace(string(out))
	if got != "target-branch" {
		t.Errorf("expected HEAD at 'target-branch', got %q", got)
	}
}
