package pr

import (
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func NewPRCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pr",
		Short:   "Manage pull requests",
		Long:    `Create, list, view, merge, close, reopen, review, comment, fetch, and checkout pull requests.`,
		Aliases: []string{"pull-request"},
	}

	cmd.PersistentFlags().StringP("repo", "R", "", "owner/repo (default: inferred from git remote)")

	cmd.AddCommand(newPRListCmd(f))
	cmd.AddCommand(newPRViewCmd(f))
	cmd.AddCommand(newPRCreateCmd(f))
	cmd.AddCommand(newPRCloseCmd(f))
	cmd.AddCommand(newPRReopenCmd(f))
	cmd.AddCommand(newPRMergeCmd(f))
	cmd.AddCommand(newPRReviewCmd(f))
	cmd.AddCommand(newPRCommentCmd(f))
	cmd.AddCommand(newPRDiffCmd(f))
	cmd.AddCommand(newPRFetchCmd(f))
	cmd.AddCommand(newPRCheckoutCmd(f))
	return cmd
}
