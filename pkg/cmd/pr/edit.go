package pr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newPREditCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title   string
		body    string
		draft   bool
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "Edit a pull request",
		Long: `Edit a pull request's title, body, or draft status.

At least one editing flag must be provided in non-interactive mode.
With no editing flags in interactive mode, the current values are loaded
and opened for editing.`,
		Example: `  gitee pr edit 42 --title "Updated title" --body "Updated description"
  gitee pr edit 42 --body ""
  gitee pr edit 42 --draft
  gitee pr edit 42 --draft=false
  gitee pr edit 42`,
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

			titleChanged := cmd.Flags().Changed("title")
			bodyChanged := cmd.Flags().Changed("body")
			draftChanged := cmd.Flags().Changed("draft")

			if !titleChanged && !bodyChanged && !draftChanged {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("at least one of --title, --body, --draft is required")
				}

				current, err := client.GetPull(f.Context, owner, repo, number)
				if err != nil {
					return fmt.Errorf("failed to fetch PR #%d: %w", number, err)
				}
				title = current.Title
				body = current.Body
				draft = current.Draft

				if f.IsTUI() {
					if err := prEditForm(&title, &body, &draft); err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
				} else {
					title, body, draft, err = promptPREdit(f, title, body, draft)
					if err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
				}

				titleChanged = true
				bodyChanged = true
				draftChanged = true
			}

			if titleChanged && strings.TrimSpace(title) == "" {
				return cmdutil.FlagErrorf("title cannot be empty")
			}

			params := &gitee.UpdatePullParams{}
			if titleChanged {
				params.Title = title
			}
			if bodyChanged {
				params.Body = &body
			}
			if draftChanged {
				params.Draft = &draft
			}

			updated, err := client.UpdatePull(f.Context, owner, repo, number, params)
			if err != nil {
				return fmt.Errorf("failed to edit PR #%d: %w", number, err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, updated)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.updated", updated.Number, updated.HTMLURL))
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New PR title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "New PR description")
	cmd.Flags().BoolVar(&draft, "draft", false, "Set or clear draft status")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func prEditForm(title, body *string, draft *bool) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(i18n.T("form.pr.title")).
				Description(i18n.T("form.pr.title_desc")).
				Value(title).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("title cannot be empty")
					}
					return nil
				}),
			huh.NewText().
				Title(i18n.T("form.pr.body")).
				Description(i18n.T("form.pr.body_desc")).
				Editor(strings.Fields(config.Editor())...).
				Value(body),
			huh.NewConfirm().
				Title(i18n.T("form.pr.draft")).
				Value(draft),
		),
	).Run()
}

func promptPREdit(f *cmdutil.Factory, title, body string, draft bool) (string, string, bool, error) {
	updatedTitle, err := cmdutil.AskInput(i18n.T("pr.edit_title_prompt"), title, false)
	if err != nil {
		return title, body, draft, err
	}

	fmt.Fprint(f.IOStreams.Out, i18n.T("pr.open_editor"))
	updatedBody, err := cmdutil.OpenEditor(f.IOStreams, "pr-body-*.md", body)
	if err != nil {
		return title, body, draft, fmt.Errorf("could not open editor: %w", err)
	}

	draftLabel := i18n.T("pr.draft_status_draft")
	readyLabel := i18n.T("pr.draft_status_ready")
	options := []string{readyLabel, draftLabel}
	if draft {
		options = []string{draftLabel, readyLabel}
	}
	status, err := cmdutil.AskSelect(i18n.T("pr.draft_status_prompt"), options, false)
	if err != nil {
		return title, body, draft, err
	}

	confirmed, err := cmdutil.AskConfirm(i18n.T("pr.confirm_edit"), false)
	if err != nil {
		return title, body, draft, err
	}
	if !confirmed {
		return title, body, draft, huh.ErrUserAborted
	}
	return updatedTitle, updatedBody, status == draftLabel, nil
}
