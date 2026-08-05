package issue

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	pkgai "gitee.com/oschina/gitee-cli/pkg/ai"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newIssueCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title    string
		body     string
		assignee string
		labels   string
		useAI    bool
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an issue",
		Long: `Create an issue in the specified repository.

With --ai, describe the issue in one sentence and the CLI will expand
it into a structured report using AI. You can then confirm, edit,
regenerate, or write manually.

Supports interactive mode (TUI) with label selection when available.`,
		Example: `  gitee issue create -t "Bug: login page crashes" -b "Steps to reproduce..."
  gitee issue create --ai
  gitee issue create -t "Title" -a alice --labels bug,urgent`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			if useAI && title == "" && body == "" {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("--ai requires an interactive terminal")
				}
				draft, err := generateIssueDraftWithAI(f)
				if err != nil {
					if cmdutil.IsUserCancelled(err) {
						return nil
					}
					fmt.Fprintf(f.IOStreams.ErrOut, "Warning: AI generation failed: %v\n", err)
				} else {
					title = draft.Title
					body = draft.Body
				}
			}

			if title == "" {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("--title is required in non-interactive mode")
				}
				if f.IsTUI() {
					repoLabels, _ := client.ListLabels(f.Context, owner, repo)
					if err := issueCreateForm(&title, &body, &assignee, &labels, repoLabels); err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
				} else {
					t, err := cmdutil.AskInput(i18n.T("issue.title_prompt"), "", false)
					if err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
					title = t
					if title == "" {
						return cmdutil.FlagErrorf("title cannot be empty")
					}
					if body == "" {
						fmt.Fprint(f.IOStreams.Out, i18n.T("issue.open_editor"))
						edited, err := cmdutil.OpenEditor(f.IOStreams, "issue-body-*.md", "")
						if err != nil {
							fmt.Fprintf(f.IOStreams.ErrOut, "Warning: could not open editor: %v\n", err)
						} else {
							body = edited
						}
					}
					if assignee == "" {
						a, err := cmdutil.AskInput(i18n.T("issue.assignee_prompt"), "", false)
						if err != nil {
							if cmdutil.IsUserCancelled(err) {
								return nil
							}
							return err
						}
						assignee = a
					}
					if labels == "" {
						l, err := cmdutil.AskInput(i18n.T("issue.labels_prompt"), "", false)
						if err != nil {
							if cmdutil.IsUserCancelled(err) {
								return nil
							}
							return err
						}
						labels = l
					}
					confirmed, err := cmdutil.AskConfirm(i18n.T("issue.confirm_create"), false)
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
				if title == "" {
					return cmdutil.FlagErrorf("title cannot be empty")
				}
			}

			iss, err := client.CreateIssue(f.Context, owner, repo, &gitee.CreateIssueParams{
				Title:    title,
				Body:     body,
				Assignee: assignee,
				Labels:   labels,
			})
			if err != nil {
				return fmt.Errorf("failed to create issue: %w", err)
			}

			if jsonOut {
				return json.NewEncoder(f.IOStreams.Out).Encode(iss)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("issue.created", iss.Number, iss.HTMLURL))
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Issue title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Issue description")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Assign to user")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated labels")
	cmd.Flags().BoolVar(&useAI, "ai", false, "Expand a short description into a structured issue using AI")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func generateIssueDraftWithAI(f *cmdutil.Factory) (*pkgai.IssueDraft, error) {
	aiCfg, err := config.AI()
	if err != nil {
		return nil, err
	}

	description, err := cmdutil.AskInput(i18n.T("issue.ai_describe"), "", f.IsTUI())
	if err != nil {
		return nil, err
	}
	if description == "" {
		return nil, fmt.Errorf("description cannot be empty")
	}

	client := pkgai.NewClient(aiCfg.BaseURL, aiCfg.Token, aiCfg.Model)
	fmt.Fprint(f.IOStreams.Out, i18n.Tf("issue.ai_expanding", aiCfg.Model))

	initial, err := pkgai.GenerateIssueDraft(f.Context, client, description, aiCfg.Language)
	if err != nil {
		return nil, err
	}

	current := &cmdutil.AIDraft{Title: initial.Title, Body: initial.Body}

	result, err := cmdutil.ConfirmAIDraft(
		f.IOStreams,
		current,
		"issue-ai-draft-*.md",
		func() (*cmdutil.AIDraft, error) {
			fmt.Fprint(f.IOStreams.Out, i18n.T("issue.ai_regenerating"))
			c2 := pkgai.NewClient(aiCfg.BaseURL, aiCfg.Token, aiCfg.Model)
			d, err := pkgai.GenerateIssueDraft(f.Context, c2, description, aiCfg.Language)
			if err != nil {
				return nil, err
			}
			return &cmdutil.AIDraft{Title: d.Title, Body: d.Body}, nil
		},
		"issue.ai_what_to_do",
		"issue.ai_use_as_is", "issue.ai_edit", "issue.ai_regenerate", "issue.ai_write_manually", "issue.ai_cancel",
		f.IsTUI(),
	)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("user chose to write manually")
	}
	return &pkgai.IssueDraft{Title: result.Title, Body: result.Body}, nil
}

func issueCreateForm(title, body, assignee, labels *string, repoLabels []gitee.Label) error {
	var selectedLabels []string
	groups := []huh.Field{
		huh.NewInput().
			Title(i18n.T("form.issue.title")).
			Description(i18n.T("form.issue.title_desc")).
			Value(title).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("title cannot be empty")
				}
				return nil
			}),
		huh.NewText().
			Title(i18n.T("form.issue.body")).
			Description(i18n.T("form.issue.body_desc")).
			Editor(strings.Fields(config.Editor())...).
			Value(body),
		huh.NewInput().
			Title(i18n.T("form.issue.assignee")).
			Description(i18n.T("form.issue.assignee_desc")).
			Value(assignee),
	}

	if len(repoLabels) > 0 {
		opts := make([]huh.Option[string], 0, len(repoLabels))
		for _, l := range repoLabels {
			opts = append(opts, huh.NewOption(l.Name, l.Name))
		}
		groups = append(groups,
			huh.NewMultiSelect[string]().
				Title(i18n.T("form.issue.labels")).
				Description(i18n.T("form.issue.labels_select")).
				Options(opts...).
				Value(&selectedLabels),
		)
	} else {
		groups = append(groups,
			huh.NewInput().
				Title(i18n.T("form.issue.labels")).
				Description(i18n.T("form.issue.labels_input")).
				Value(labels),
		)
	}

	if err := huh.NewForm(huh.NewGroup(groups...)).Run(); err != nil {
		return err
	}

	if len(selectedLabels) > 0 {
		*labels = strings.Join(selectedLabels, ",")
	}
	return nil
}
