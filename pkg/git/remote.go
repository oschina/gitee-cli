package git

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// remoteURLPattern matches the git remote URL forms Gitee supports and
// captures the host and the repository path.
//
// Supported forms:
//
//	https://gitee.com/<owner>/<repo>.git
//	https://<user>@gitee.com/<owner>/<repo>.git   (userinfo is ignored)
//	git@gitee.com:<owner>/<repo>.git
//	git@gitee.com:<ent>/<group>/<repo>.git        (multi-segment namespace)
//
// The repository path accepts any number of "/" separated segments so that
// enterprise "project group" repositories (<ent>/<group>/<repo>) are
// recognised as well.
var remoteURLPattern = regexp.MustCompile(`(?:https?://|git@)(?:[^@/]+@)?([a-zA-Z0-9.-]+)[:/](.+?)(?:\.git)?$`)

// ErrNotGitRepo reports that the current directory is not inside a git
// repository (or git is unavailable).
var ErrNotGitRepo = errors.New("not a git repository (or git is not installed)")

// ErrUnparseableRemote reports that the repository has git remotes, but none
// of their URLs could be parsed into an owner/repository pair.
var ErrUnparseableRemote = errors.New("could not parse owner/repo from git remote")

type Remote struct {
	Name string
	URL  string
	Host string
}

func GiteeRemotes() ([]Remote, error) {
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotGitRepo, err)
	}

	seenRemote := map[string]bool{}
	seenUnparsed := map[string]bool{}
	var remotes []Remote
	var unparsed []string
	recordUnparsed := func(url string) {
		if seenUnparsed[url] {
			return
		}
		seenUnparsed[url] = true
		unparsed = append(unparsed, redactRemoteURL(url))
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name, url := fields[0], fields[1]
		if seenRemote[name] {
			continue
		}
		m := remoteURLPattern.FindStringSubmatch(url)
		if m == nil {
			recordUnparsed(url)
			continue
		}
		if _, _, err := SplitRepoPath(m[2]); err != nil {
			recordUnparsed(url)
			continue
		}
		seenRemote[name] = true
		remotes = append(remotes, Remote{Name: name, URL: url, Host: m[1]})
	}
	if len(remotes) == 0 {
		if len(unparsed) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrUnparseableRemote, strings.Join(unparsed, ", "))
		}
		return nil, fmt.Errorf("no git remote found in this repository")
	}
	return remotes, nil
}

func RepoFromRemote() (owner, repo string, err error) {
	remotes, err := GiteeRemotes()
	if err != nil {
		return "", "", err
	}
	r := remotes[0]
	return parseRemoteURL(r.URL)
}

func FetchPR(remoteURL string, number int, localBranch string) error {
	refspec := fmt.Sprintf("pull/%d/head:%s", number, localBranch)
	cmd := exec.Command("git", "fetch", remoteURL, refspec)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func parseRemoteURL(rawURL string) (owner, repo string, err error) {
	m := remoteURLPattern.FindStringSubmatch(rawURL)
	if m == nil {
		return "", "", fmt.Errorf("cannot parse owner/repo from remote URL %q", rawURL)
	}
	return SplitRepoPath(m[2])
}

// SplitRepoPath splits an owner/repository path into its owner and repository
// parts. The repository is the last "/" separated segment; everything before
// it is the owner. This keeps enterprise three-segment paths
// (<ent>/<group>/<repo>) consistent across every API call: read endpoints
// rebuild "<owner>/<repo>" while create endpoints pass the multi-segment
// "<owner>" as the namespace and "<repo>" as the repository field.
func SplitRepoPath(s string) (owner, repo string, err error) {
	path := strings.Trim(s, "/")
	idx := strings.LastIndex(path, "/")
	if idx <= 0 || idx == len(path)-1 {
		return "", "", fmt.Errorf("invalid format %q, expected owner/repo", s)
	}
	return path[:idx], path[idx+1:], nil
}

// redactRemoteURL strips credentials from the userinfo portion of an HTTP(S)
// remote URL so that unparseable URLs echoed in error messages cannot leak
// embedded tokens. The SCP-style "git@host:path" form is left untouched
// because "git" there is a protocol username, not a secret.
func redactRemoteURL(rawURL string) string {
	i := strings.Index(rawURL, "://")
	if i < 0 {
		return rawURL
	}
	rest := rawURL[i+3:]
	at := strings.Index(rest, "@")
	if at < 0 {
		return rawURL
	}
	if slash := strings.Index(rest, "/"); slash >= 0 && slash < at {
		return rawURL
	}
	return rawURL[:i+3] + "***@" + rest[at+1:]
}

func DefaultBranch(remote string) string {
	if remote == "" {
		remote = "origin"
	}
	out, err := exec.Command("git", "symbolic-ref", "--short", "refs/remotes/"+remote+"/HEAD").Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		if idx := strings.LastIndex(branch, "/"); idx >= 0 {
			return branch[idx+1:]
		}
		return branch
	}
	for _, candidate := range []string{"main", "master"} {
		check := exec.Command("git", "rev-parse", "--verify", remote+"/"+candidate)
		if check.Run() == nil {
			return candidate
		}
	}
	return "master"
}

func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func LocalBranches() ([]string, error) {
	out, err := exec.Command("git", "branch", "--format=%(refname:short)").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list local branches: %w", err)
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

func Checkout(branch string) error {
	out, err := exec.Command("git", "checkout", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func BranchExists(branch string) (bool, error) {
	branches, err := LocalBranches()
	if err != nil {
		return false, err
	}
	for _, b := range branches {
		if b == branch {
			return true, nil
		}
	}
	return false, nil
}

func DiffBranch(base, head string) (string, error) {
	out, err := exec.Command("git", "diff", base+"..."+head).Output()
	if err != nil {
		out2, _ := exec.Command("git", "diff", "HEAD").Output()
		return strings.TrimSpace(string(out2)), nil
	}
	return strings.TrimSpace(string(out)), nil
}

func LogBranch(base, head string) (string, error) {
	out, err := exec.Command("git", "log", "--oneline", base+".."+head).Output()
	if err != nil {
		out2, _ := exec.Command("git", "log", "--oneline", "-10").Output()
		return strings.TrimSpace(string(out2)), nil
	}
	return strings.TrimSpace(string(out)), nil
}

// RepoRoot returns the absolute path of the current git repository root.
func RepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repo: %w", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("could not determine repo root")
	}
	return root, nil
}
