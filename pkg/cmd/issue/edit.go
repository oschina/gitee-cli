package issue

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newIssueEditCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title    string
		body     string
		assignee string
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "Edit an issue",
		Long: `Edit an issue's title, body, or assignee.

At least one of --title, --body, or --assignee must be provided in
non-interactive mode. In interactive mode, fetches the current values
and opens them for editing.`,
		Example: `  gitee issue edit ICX4FO -t "New title" -b "Updated description"
  gitee issue edit ICX4FO -a alice
  gitee issue edit ICX4FO     # interactive mode`,
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

			titleChanged := cmd.Flags().Changed("title")
			bodyChanged := cmd.Flags().Changed("body")
			assigneeChanged := cmd.Flags().Changed("assignee")

			if !titleChanged && !bodyChanged && !assigneeChanged {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("at least one of --title, --body, --assignee is required")
				}

				iss, err := client.GetIssue(f.Context, owner, repo, number)
				if err != nil {
					return fmt.Errorf("failed to fetch issue %s: %w", number, err)
				}
				title = iss.Title
				body = iss.Body

				if f.IsTUI() {
					if err := issueEditForm(&title, &body, &assignee); err != nil {
						return err
					}
				} else {
					t, err := cmdutil.AskInput(i18n.T("issue.edit_title_prompt"), title, false)
					if err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
					if strings.TrimSpace(t) != "" {
						title = t
					}
					fmt.Fprint(f.IOStreams.Out, i18n.T("issue.open_editor"))
					edited, err := cmdutil.OpenEditor(f.IOStreams, "issue-body-*.md", body)
					if err != nil {
						fmt.Fprintf(f.IOStreams.ErrOut, "Warning: could not open editor: %v\n", err)
					} else {
						body = edited
					}
					a, err := cmdutil.AskInput(i18n.T("issue.assignee_prompt"), "", false)
					if err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
					if strings.TrimSpace(a) != "" {
						assignee = a
					}
					confirmed, err := cmdutil.AskConfirm(i18n.T("issue.confirm_edit"), false)
					if err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
					if !confirmed {
						return nil
					}
				}
			}

			params := &gitee.UpdateIssueParams{}
			if title != "" {
				params.Title = title
			}
			if body != "" {
				params.Body = body
			}
			if assignee != "" {
				params.Assignee = assignee
			}

			iss, err := client.UpdateIssue(f.Context, owner, repo, number, params)
			if err != nil {
				return fmt.Errorf("failed to edit issue %s: %w", number, err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, iss)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("issue.updated", iss.Number, iss.HTMLURL))
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "New description")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "New assignee")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func issueEditForm(title, body, assignee *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(i18n.T("form.issue.title")).
				Value(title),
			huh.NewText().
				Title(i18n.T("form.issue.body")).
				Editor(strings.Fields(config.Editor())...).
				Value(body),
			huh.NewInput().
				Title(i18n.T("form.issue.assignee")).
				Value(assignee),
		),
	).Run()
}
