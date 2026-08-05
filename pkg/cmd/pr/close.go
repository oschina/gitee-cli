package pr

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newPRCloseCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:     "close <number>",
		Short:   "Close a pull request without merging",
		Long:    `Close a pull request without merging it. The PR will be marked as "closed" and can be reopened later with 'pr reopen'.`,
		Example: `  gitee pr close 42`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
			owner, repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			_, err = client.UpdatePull(f.Context, owner, repo, number, &gitee.UpdatePullParams{
				State: "closed",
			})
			if err != nil {
				return fmt.Errorf("failed to close PR #%d: %w", number, err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, cmdutil.StatusResult{Success: true})
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.closed", number))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
