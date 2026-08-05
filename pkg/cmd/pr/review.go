package pr

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	pkgai "gitee.com/oschina/gitee-cli/pkg/ai"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newPRReviewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		useAI bool
		force bool
		body  string
	)

	cmd := &cobra.Command{
		Use:   "review <number>",
		Short: "Approve a pull request, or analyse it with AI",
		Long: `Review a pull request.

Without --ai, the command approves the PR (or leaves a comment via --body).
Use --force to approve as admin even when review requirements are not met.
Use --ai to fetch the diff and get an AI-generated code review.`,
		Example: `  gitee pr review 42                     # approve
  gitee pr review 42 -b "Looks good!"    # approve with comment
  gitee pr review 42 --ai                # AI code review
  gitee pr review 42 --force             # admin force approve`,
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

			if useAI {
				return runAIReview(f, owner, repo, number)
			}

			bodyChanged := cmd.Flags().Changed("body")

			if !bodyChanged && f.IOStreams.IsStdinTerminal() {
				action, err := chooseReviewAction(f)
				if err != nil {
					if cmdutil.IsUserCancelled(err) {
						return nil
					}
					return err
				}

				switch action {
				case i18n.T("pr.review_approve"):
					return approveAndMaybeComment(f, client, owner, repo, number, force, "")
				case i18n.T("pr.review_approve_comment"):
					b, err := cmdutil.OpenEditor(f.IOStreams, "pr-review-*.md", "")
					if err != nil {
						fmt.Fprintf(f.IOStreams.ErrOut, "Warning: could not open editor: %v\n", err)
					}
					return approveAndMaybeComment(f, client, owner, repo, number, force, b)
				case i18n.T("pr.review_comment_only"):
					b, err := cmdutil.OpenEditor(f.IOStreams, "pr-review-*.md", "")
					if err != nil {
						return fmt.Errorf("could not open editor: %w", err)
					}
					return commentOnly(f, client, owner, repo, number, b)
				}
				return nil
			}

			return approveAndMaybeComment(f, client, owner, repo, number, force, body)
		},
	}

	cmd.Flags().BoolVar(&useAI, "ai", false, "Analyse PR diff with AI and display review in pager")
	cmd.Flags().BoolVar(&force, "force", false, "Force approve even if review requirements are not met (admin only)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Leave a comment on the PR after approving")
	return cmd
}

func chooseReviewAction(f *cmdutil.Factory) (string, error) {
	return cmdutil.AskSelect(i18n.T("pr.review_action_prompt"), []string{
		i18n.T("pr.review_approve"),
		i18n.T("pr.review_approve_comment"),
		i18n.T("pr.review_comment_only"),
	}, f.IsTUI())
}

func approveAndMaybeComment(f *cmdutil.Factory, client *gitee.Client, owner, repo string, number int, force bool, body string) error {
	params := &gitee.ReviewPullParams{Force: force}
	if err := client.ReviewPull(f.Context, owner, repo, number, params); err != nil {
		return fmt.Errorf("failed to approve PR #%d: %w", number, err)
	}
	fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.approved", number))

	if body != "" {
		comment, err := client.CreatePullComment(f.Context, owner, repo, number, body)
		if err != nil {
			return fmt.Errorf("PR #%d was approved, but the requested comment failed: %w", number, err)
		}
		fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.comment_added", comment.ID, number))
	}
	return nil
}

func commentOnly(f *cmdutil.Factory, client *gitee.Client, owner, repo string, number int, body string) error {
	if body == "" {
		return cmdutil.FlagErrorf("comment body cannot be empty")
	}
	comment, err := client.CreatePullComment(f.Context, owner, repo, number, body)
	if err != nil {
		return fmt.Errorf("failed to comment on PR #%d: %w", number, err)
	}
	fmt.Fprint(f.IOStreams.Out, i18n.Tf("pr.comment_added", comment.ID, number))
	return nil
}

func runAIReview(f *cmdutil.Factory, owner, repo string, number int) error {
	aiCfg, err := config.AI()
	if err != nil {
		return err
	}

	giteeClient, err := f.GiteeClient()
	if err != nil {
		return err
	}

	fmt.Fprintf(f.IOStreams.Out, "✦ Fetching PR #%d...\n", number)
	pr, err := giteeClient.GetPull(f.Context, owner, repo, number)
	if err != nil {
		return fmt.Errorf("failed to get PR: %w", err)
	}

	fmt.Fprintln(f.IOStreams.Out, "✦ Fetching diff...")
	diff, err := giteeClient.GetPullDiff(f.Context, owner, repo, number)
	if err != nil {
		return fmt.Errorf("failed to get PR diff: %w", err)
	}

	const maxDiffBytes = 16000
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n... (truncated)"
	}

	aiClient := pkgai.NewClient(aiCfg.BaseURL, aiCfg.Token, aiCfg.Model)
	fmt.Fprintf(f.IOStreams.Out, "✦ Analysing with %s...\n", aiCfg.Model)

	review, err := pkgai.GeneratePRReview(f.Context, aiClient, pr.Title, pr.Body, diff, aiCfg.Language)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("AI Review — PR #%d: %s", pr.Number, pr.Title)
	if f.IsTUI() {
		return tui.RunPager(title, review, tui.ContentMarkdown)
	}
	fmt.Fprintf(f.IOStreams.Out, "\n%s\n%s\n\n%s\n", title,
		"════════════════════════════════════════", review)
	return nil
}
