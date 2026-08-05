package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newIssueViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "View an issue",
		Long:  `View the details of a specific issue, including its title, body, state, assignee, and labels.`,
		Example: `  gitee issue view ICX4FO
  gitee issue view ICX4FO --json=number,title,state`,
		Args: cobra.ExactArgs(1),
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

			iss, err := client.GetIssue(f.Context, owner, repo, number)
			if err != nil {
				return fmt.Errorf("failed to get issue %s: %w", number, err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Issue](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, iss)
				}
				result, err := cmdutil.SelectFields(iss, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			if f.IsTUI() {
				title := fmt.Sprintf("#%s %s [%s]", iss.Number, iss.Title, iss.State)
				body := iss.Body
				if body == "" {
					body = "_No description provided._"
				}
				return tui.RunPager(title, body, tui.ContentMarkdown)
			}

			fmt.Fprintf(f.IOStreams.Out, "#%s %s\n", iss.Number, iss.Title)
			fmt.Fprintf(f.IOStreams.Out, "State: %s\n", iss.State)
			fmt.Fprintf(f.IOStreams.Out, "URL:   %s\n", iss.HTMLURL)
			if iss.Body != "" {
				fmt.Fprintf(f.IOStreams.Out, "\n%s\n", iss.Body)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Issue]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
