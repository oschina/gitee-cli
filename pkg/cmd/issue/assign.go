package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newIssueAssignCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "assign <number> <assignee>",
		Short: "Assign an issue to a user",
		Long: `Assign an issue to a user by username.

The assignee is the Gitee login name, not the display name.`,
		Example: `  gitee issue assign ICX4FO alice`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, assignee := args[0], args[1]
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			_, err = client.UpdateIssue(f.Context, owner, repo, number, &gitee.UpdateIssueParams{
				Assignee: &assignee,
			})
			if err != nil {
				return fmt.Errorf("failed to assign issue %s: %w", number, err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, cmdutil.StatusResult{Success: true})
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("issue.assigned", number, assignee))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
