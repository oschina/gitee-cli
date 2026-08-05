package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var remoteURLPattern = regexp.MustCompile(`(?:https?://|git@)([a-zA-Z0-9.-]+)[:/]([^/]+/[^/]+?)(?:\.git)?$`)

type Remote struct {
	Name string
	URL  string
	Host string
}

func GiteeRemotes() ([]Remote, error) {
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repo or git not found: %w", err)
	}

	seen := map[string]bool{}
	var remotes []Remote
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name, url := fields[0], fields[1]
		m := remoteURLPattern.FindStringSubmatch(url)
		if m == nil {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		remotes = append(remotes, Remote{Name: name, URL: url, Host: m[1]})
	}
	if len(remotes) == 0 {
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
	parts := strings.SplitN(m[2], "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cannot parse owner/repo from remote URL %q", rawURL)
	}
	return parts[0], parts[1], nil
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
