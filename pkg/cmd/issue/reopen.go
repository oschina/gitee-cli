package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newIssueReopenCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:     "reopen <number>",
		Short:   "Reopen a closed issue",
		Long:    `Reopen a previously closed issue. The issue state will be set to "open".`,
		Example: `  gitee issue reopen ICX4FO`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number := args[0]
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			_, err = client.UpdateIssue(f.Context, owner, repo, number, &gitee.UpdateIssueParams{
				State: "open",
			})
			if err != nil {
				return fmt.Errorf("failed to reopen issue %s: %w", number, err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, cmdutil.StatusResult{Success: true})
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("issue.reopened", number))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
