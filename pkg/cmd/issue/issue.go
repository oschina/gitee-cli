package issue

import (
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func NewIssueCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
		Long:  `Create, list, view, edit, close, reopen, assign, and comment on issues.`,
	}

	cmd.PersistentFlags().StringP("repo", "R", "", "owner/repo (default: inferred from git remote)")

	cmd.AddCommand(newIssueListCmd(f))
	cmd.AddCommand(newIssueViewCmd(f))
	cmd.AddCommand(newIssueCreateCmd(f))
	cmd.AddCommand(newIssueCloseCmd(f))
	cmd.AddCommand(newIssueReopenCmd(f))
	cmd.AddCommand(newIssueEditCmd(f))
	cmd.AddCommand(newIssueAssignCmd(f))
	cmd.AddCommand(newIssueCommentCmd(f))
	return cmd
}
