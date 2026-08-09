package pr

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newPRDiffCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "diff <number>",
		Short: "View the diff of a pull request",
		Long: `View the full diff of a pull request. Shows all changed files with
additions and deletions highlighted.

In TUI mode, the diff is displayed in a pager with syntax coloring.
In non-TUI mode, the raw diff is printed to stdout.

Use --json to output the changed files as structured JSON instead of a
unified diff.`,
		Example: `  gitee pr diff 42
  gitee pr diff 42 --json=filename,status,additions,deletions`,
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

			files, err := client.GetPullDiffFiles(f.Context, owner, repo, number)
			if err != nil {
				return fmt.Errorf("failed to get diff for PR #%d: %w", number, err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.DiffFile](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, files)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, files, fields)
			}

			content := formatDiffFiles(files)

			if f.IsTUI() {
				title := fmt.Sprintf("PR #%d diff", number)
				return tui.RunPager(title, content, tui.ContentDiff)
			}

			fmt.Fprint(f.IOStreams.Out, content)
			return nil
		},
	}

	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.DiffFile]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
