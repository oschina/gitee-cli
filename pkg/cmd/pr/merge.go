package pr

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newPRMergeCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		mergeMethod       string
		pruneSourceBranch bool
		jsonOut           bool
	)

	cmd := &cobra.Command{
		Use:   "merge <number>",
		Short: "Merge a pull request",
		Long: `Merge a pull request. Supports three merge methods:
- merge: standard merge commit (default)
- squash: squash all commits into one
- rebase: rebase onto base branch

Use --delete-branch to prune the source branch after merging.`,
		Example: `  gitee pr merge 42
  gitee pr merge 42 --method squash
  gitee pr merge 42 --method rebase --delete-branch`,
		Args: cobra.ExactArgs(1),
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

			if err := client.MergePull(f.Context, owner, repo, number, mergeMethod, pruneSourceBranch); err != nil {
				return fmt.Errorf("failed to merge PR #%d: %w", number, err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, cmdutil.StatusResult{Success: true})
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.merged", number))
			return nil
		},
	}

	cmd.Flags().StringVarP(&mergeMethod, "method", "m", "merge", "Merge method: merge, squash, rebase")
	cmd.Flags().BoolVar(&pruneSourceBranch, "delete-branch", false, "Delete source branch after merge")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
