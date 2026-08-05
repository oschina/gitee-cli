package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newIssueCommentCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		body    string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "comment <number>",
		Short: "Add a comment to an issue",
		Long:  `Add a comment to an issue. Opens an editor for the comment body if --body is not provided.`,
		Example: `  gitee issue comment ICX4FO -b "Fixed in latest release"
  gitee issue comment ICX4FO     # opens editor`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number := args[0]
			if body == "" {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("--body is required in non-interactive mode")
				}
				edited, err := cmdutil.OpenEditor(f.IOStreams, "issue-comment-*.md", "")
				if err != nil {
					return fmt.Errorf("could not open editor: %w", err)
				}
				body = edited
			}
			if body == "" {
				return cmdutil.FlagErrorf("comment body cannot be empty")
			}
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			comment, err := client.CreateIssueComment(f.Context, owner, repo, number, body)
			if err != nil {
				return fmt.Errorf("failed to comment on issue %s: %w", number, err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, comment)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("issue.comment_added", comment.ID, number))
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
