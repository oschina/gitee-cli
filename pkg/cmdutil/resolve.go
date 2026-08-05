package cmdutil

import (
	"fmt"
	"strings"

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

	defaultRepo := config.DefaultRepo()
	if defaultRepo != "" {
		return ParseOwnerRepo(defaultRepo)
	}

	return "", "", err
}

func ParseOwnerRepo(s string) (owner, repo string, err error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid format %q, expected owner/repo", s)
	}
	return parts[0], parts[1], nil
}
