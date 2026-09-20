package cmdutil

import (
	"errors"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/git"
)

func ResolveRepo(cmd *cobra.Command) (owner, repo string, err error) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	if repoFlag != "" {
		return ParseOwnerRepo(repoFlag)
	}

	owner, repo, err = git.RepoFromRemote()
	if err == nil {
		return owner, repo, nil
	}

	// Remotes that exist but cannot be parsed mean the repository was spelled
	// in a way we do not understand. Falling back to default_repo here would
	// silently run the command against an unrelated repository.
	if errors.Is(err, git.ErrUnparseableRemote) {
		return "", "", err
	}

	defaultRepo := config.DefaultRepo()
	if defaultRepo != "" {
		return ParseOwnerRepo(defaultRepo)
	}

	return "", "", err
}

func ParseOwnerRepo(s string) (owner, repo string, err error) {
	return git.SplitRepoPath(s)
}
