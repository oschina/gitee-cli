package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	pkgai "gitee.com/oschina/gitee-cli/pkg/ai"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/git"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newPRCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title      string
		body       string
		base       string
		head       string
		draft      bool
		assignees  string
		testers    string
		useAI      bool
		noTemplate bool
		jsonOut    bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		Long: `Create a pull request in the current repository.

By default, the head branch is the current branch and the base branch
is inferred from the git remote (falls back to master).

With --ai, the CLI reads your git diff and commit log to generate a
title and description using AI, then lets you confirm, edit, or
regenerate before creating.

Supports PR templates from .gitee/PULL_REQUEST_TEMPLATE.md or
the local config file. Use --no-template to skip.`,
		Example: `  gitee pr create                       # interactive mode
  gitee pr create -t "Title" -b "Body"
  gitee pr create --ai                  # AI-generated description
  gitee pr create --base main --head feature-branch
	  gitee pr create --assignees alice,bob --testers carol`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			if head == "" {
				head, err = git.CurrentBranch()
				if err != nil {
					return fmt.Errorf("could not detect current branch, use --head: %w", err)
				}
			}
			if base == "" {
				base = git.DefaultBranch("")
			}
			if title == "" && f.IOStreams.IsStdinTerminal() && !f.IsTUI() && !cmd.Flags().Changed("base") {
				base, err = promptPRBaseBranch(head, base, cmdutil.AskInput)
				if err != nil {
					if cmdutil.IsUserCancelled(err) {
						return nil
					}
					return err
				}
			}

			var tmpl string
			if body == "" && !noTemplate {
				tmpl = loadPRTemplate(f.Context, client, owner, repo, base)
			}

			if useAI && title == "" && body == "" {
				draft, err := generatePRDraftWithAI(f, base, head, tmpl)
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
					if err := prCreateForm(&title, &body, &base, &draft, &assignees, &testers, head, tmpl); err != nil {
						if cmdutil.IsUserCancelled(err) {
							return nil
						}
						return err
					}
				} else {
					t, err := cmdutil.AskInput(i18n.T("pr.title_prompt"), "", false)
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
						fmt.Fprint(f.IOStreams.Out, i18n.T("pr.open_editor"))
						edited, err := cmdutil.OpenEditor(f.IOStreams, "pr-body-*.md", tmpl)
						if err != nil {
							fmt.Fprintf(f.IOStreams.ErrOut, "Warning: could not open editor: %v\n", err)
						} else {
							body = edited
						}
					}
					if assignees == "" {
						a, err := cmdutil.AskInput(i18n.T("pr.assignees_prompt"), "", false)
						if err != nil {
							if cmdutil.IsUserCancelled(err) {
								return nil
							}
							return err
						}
						assignees = a
					}
					if testers == "" {
						t, err := cmdutil.AskInput(i18n.T("pr.testers_prompt"), "", false)
						if err != nil {
							if cmdutil.IsUserCancelled(err) {
								return nil
							}
							return err
						}
						testers = t
					}
					confirmed, err := cmdutil.AskConfirm(i18n.T("pr.confirm_create"), false)
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

			params := &gitee.CreatePullParams{
				Title: title,
				Body:  body,
				Head:  head,
				Base:  base,
				Draft: draft,
			}
			if assignees != "" {
				params.Assignees = assignees
			}
			if testers != "" {
				params.Testers = testers
			}

			pr, err := client.CreatePull(f.Context, owner, repo, params)
			if err != nil {
				return fmt.Errorf("failed to create PR: %w", err)
			}

			if jsonOut {
				return json.NewEncoder(f.IOStreams.Out).Encode(pr)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.created", pr.Number, pr.HTMLURL))
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "PR title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "PR description")
	cmd.Flags().StringVar(&base, "base", "", "Base branch (default: inferred from git remote, falls back to master)")
	cmd.Flags().StringVar(&head, "head", "", "Head branch (default: current branch)")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create as draft PR")
	cmd.Flags().StringVar(&assignees, "assignees", "", "Comma-separated reviewer usernames")
	cmd.Flags().StringVar(&testers, "testers", "", "Comma-separated tester usernames")
	cmd.Flags().BoolVar(&useAI, "ai", false, "Generate title and body using AI (requires ai.token and ai.base_url config)")
	cmd.Flags().BoolVar(&noTemplate, "no-template", false, "Skip PR description template")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func generatePRDraftWithAI(f *cmdutil.Factory, base, head, tmpl string) (*pkgai.PRDraft, error) {
	aiCfg, err := config.AI()
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(f.IOStreams.Out, "✦ Collecting git context...")

	diff, err := git.DiffBranch(base, head)
	if err != nil || diff == "" {
		diff = "(no diff available)"
	}

	const maxDiffBytes = 12000
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n... (truncated)"
	}

	commits, _ := git.LogBranch(base, head)

	client := pkgai.NewClient(aiCfg.BaseURL, aiCfg.Token, aiCfg.Model)
	fmt.Fprintf(f.IOStreams.Out, "✦ Generating PR description with %s...\n", aiCfg.Model)

	initial, err := pkgai.GeneratePRDraft(f.Context, client, diff, commits, aiCfg.Language, tmpl)
	if err != nil {
		return nil, err
	}

	current := &cmdutil.AIDraft{Title: initial.Title, Body: initial.Body}

	result, err := cmdutil.ConfirmAIDraft(
		f.IOStreams,
		current,
		"pr-ai-draft-*.md",
		func() (*cmdutil.AIDraft, error) {
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.ai_regenerating", aiCfg.Model))
			c2 := pkgai.NewClient(aiCfg.BaseURL, aiCfg.Token, aiCfg.Model)
			d, err := pkgai.GeneratePRDraft(f.Context, c2, diff, commits, aiCfg.Language, tmpl)
			if err != nil {
				return nil, err
			}
			return &cmdutil.AIDraft{Title: d.Title, Body: d.Body}, nil
		},
		"pr.ai_what_to_do",
		"pr.ai_use_as_is", "pr.ai_edit", "pr.ai_regenerate", "pr.ai_write_manually", "pr.ai_cancel",
		f.IsTUI(),
	)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("user chose to write manually")
	}
	return &pkgai.PRDraft{Title: result.Title, Body: result.Body}, nil
}

func promptPRBaseBranch(head, base string, askInput func(string, string, bool) (string, error)) (string, error) {
	return askInput(i18n.Tf("pr.base_prompt", head), base, false)
}

func prCreateForm(title, body, base *string, draft *bool, assignees, testers *string, currentHead, tmpl string) error {
	if *body == "" && tmpl != "" {
		*body = tmpl
	}
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
			huh.NewInput().
				Title(i18n.T("form.pr.base")).
				Description(i18n.Tf("form.pr.base_desc", currentHead)).
				Value(base),
		),
		huh.NewGroup(
			huh.NewInput().
				Title(i18n.T("form.pr.assignees")).
				Description(i18n.T("form.pr.assignees_desc")).
				Value(assignees),
			huh.NewInput().
				Title(i18n.T("form.pr.testers")).
				Description(i18n.T("form.pr.testers_desc")).
				Value(testers),
			huh.NewConfirm().
				Title(i18n.T("form.pr.draft")).
				Value(draft),
		),
	).Run()
}

func loadPRTemplate(ctx context.Context, client *gitee.Client, owner, repo, base string) string {
	tmpl, err := client.GetFileContent(ctx, owner, repo, ".gitee/PULL_REQUEST_TEMPLATE.md", base)
	if err == nil && tmpl != "" {
		return tmpl
	}

	data, err := os.ReadFile(config.PRTemplatePath())
	if err == nil {
		return strings.TrimRight(string(data), "\n")
	}

	return ""
}
