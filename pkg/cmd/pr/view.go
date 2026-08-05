package pr

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newPRViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "View a pull request",
		Long:  `View the details of a specific pull request, including its title, body, branch, author, and state.`,
		Example: `  gitee pr view 42
  gitee pr view 42 --json=number,title,state,head.ref,base.ref`,
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

			pr, err := client.GetPull(f.Context, owner, repo, number)
			if err != nil {
				return fmt.Errorf("failed to get PR #%d: %w", number, err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.PullRequest](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, pr)
				}
				result, err := cmdutil.SelectFields(pr, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			if f.IsTUI() {
				title := fmt.Sprintf("#%d %s (%s → %s)", pr.Number, pr.Title, pr.Head.Ref, pr.Base.Ref)
				body := pr.Body
				if body == "" {
					body = "_No description provided._"
				}
				return tui.RunPager(title, body, tui.ContentMarkdown)
			}

			fmt.Fprintf(f.IOStreams.Out, "#%d %s\n", pr.Number, pr.Title)
			fmt.Fprintf(f.IOStreams.Out, "State:  %s\n", pr.State)
			fmt.Fprintf(f.IOStreams.Out, "Author: %s\n", pr.User.Login)
			fmt.Fprintf(f.IOStreams.Out, "Branch: %s → %s\n", pr.Head.Ref, pr.Base.Ref)
			fmt.Fprintf(f.IOStreams.Out, "URL:    %s\n", pr.HTMLURL)
			if pr.Body != "" {
				fmt.Fprintf(f.IOStreams.Out, "\n%s\n", pr.Body)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.PullRequest]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
