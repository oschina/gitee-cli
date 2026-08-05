package pr

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	gitpkg "gitee.com/oschina/gitee-cli/pkg/git"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newPRListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		state           string
		limit           int
		page            int
		head            string
		base            string
		sort            string
		since           string
		direction       string
		milestoneNumber int
		labels          string
		author          string
		assignee        string
		tester          string
		jsonFields      string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List pull requests",
		Long:    `List pull requests in the repository. Supports filtering by state, branch, author, assignee, labels, and more.`,
		Aliases: []string{"ls"},
		Example: `  gitee pr list
  gitee pr list -s open
  gitee pr list --author alice --assignee bob
  gitee pr list --labels bug,performance
  gitee pr list --json=number,title,state,user.login`,
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

			prs, err := client.ListPulls(f.Context, owner, repo, &gitee.ListPullsParams{
				State:           state,
				Head:            head,
				Base:            base,
				Sort:            sort,
				Since:           since,
				Direction:       direction,
				MilestoneNumber: milestoneNumber,
				Labels:          labels,
				Page:            page,
				PerPage:         limit,
				Author:          author,
				Assignee:        assignee,
				Tester:          tester,
			})
			if err != nil {
				return fmt.Errorf("failed to list pull requests: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.PullRequest](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, prs)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, prs, fields)
			}

			if len(prs) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("pr.no_results"))
				return nil
			}

			if f.IsTUI() {
				return prListTUI(f.Context, prs, owner, repo, f.Hostname, client)
			}

			w := tabwriter.NewWriter(f.IOStreams.Out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "#\tTITLE\tAUTHOR\tSTATE\tHEAD\tBASE\n")
			for _, pr := range prs {
				title := tui.Truncate(pr.Title, 50)
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", pr.Number, title, pr.User.Login, pr.State, pr.Head.Ref, pr.Base.Ref)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVarP(&state, "state", "s", "open", "Filter by state: open, closed, merged, all")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of PRs per page")
	cmd.Flags().IntVar(&limit, "per-page", 20, "Number of PRs per page")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().StringVar(&head, "head", "", "Filter by head branch")
	cmd.Flags().StringVar(&base, "base", "", "Filter by base branch")
	cmd.Flags().StringVar(&sort, "sort", "created", "Sort by field")
	cmd.Flags().StringVar(&since, "since", "", "Filter by updated since time (ISO 8601)")
	cmd.Flags().StringVar(&direction, "direction", "desc", "Sort direction: asc, desc")
	cmd.Flags().IntVar(&milestoneNumber, "milestone-number", 0, "Filter by milestone number")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&author, "author", "", "Filter by PR author username")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Filter by assignee username")
	cmd.Flags().StringVar(&tester, "tester", "", "Filter by tester username")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.PullRequest]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func formatDiffFiles(files []gitee.DiffFile) string {
	if len(files) == 0 {
		return "No changed files."
	}
	var sb strings.Builder
	for _, f := range files {
		status := ""
		if f.Status != nil {
			status = *f.Status
		}
		sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", f.Patch.OldPath, f.Patch.NewPath))
		if f.Patch.NewFile {
			sb.WriteString("new file\n")
		} else if f.Patch.DeletedFile {
			sb.WriteString("deleted file\n")
		} else if f.Patch.RenamedFile {
			sb.WriteString(fmt.Sprintf("renamed from %s\n", f.Patch.OldPath))
		}
		if status != "" {
			sb.WriteString(fmt.Sprintf("status: %s  +%s -%s\n", status, f.Additions, f.Deletions))
		} else {
			sb.WriteString(fmt.Sprintf("+%s -%s\n", f.Additions, f.Deletions))
		}
		if f.Patch.TooLarge {
			sb.WriteString("(patch too large to display)\n")
		} else if f.Patch.Diff != "" {
			sb.WriteString(f.Patch.Diff)
			if !strings.HasSuffix(f.Patch.Diff, "\n") {
				sb.WriteByte('\n')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func prListTUI(ctx context.Context, prs []gitee.PullRequest, owner, repo, hostname string, client *gitee.Client) error {
	columns := []table.Column{
		{Title: "#", Width: 6},
		{Title: "Title", Width: 40},
		{Title: "Author", Width: 15},
		{Title: "State", Width: 8},
		{Title: "Head", Width: 25},
		{Title: "Base", Width: 15},
	}

	rows := make([]table.Row, 0, len(prs))
	for _, pr := range prs {
		title := tui.Truncate(pr.Title, 38)
		rows = append(rows, table.Row{
			strconv.Itoa(pr.Number), title, pr.User.Login, pr.State, pr.Head.Ref, pr.Base.Ref,
		})
	}

	return tui.RunTable(tui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Height:  min(len(prs)+1, 20),
		HelpKeys: []tui.HelpKey{
			{Key: "enter", Desc: "open"},
			{Key: "v", Desc: "preview"},
			{Key: "c", Desc: "copy number"},
			{Key: "d", Desc: "diff"},
			{Key: "f", Desc: "fetch"},
			{Key: "q", Desc: "quit"},
		},
		OnSelect: func(row table.Row) {
			if hostname == "" {
				hostname = config.DefaultHost
			}
			url := fmt.Sprintf("https://%s/%s/%s/pulls/%s", hostname, owner, repo, row[0])
			browser.OpenURL(url)
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(row[0])
		},
		OnView: func(row table.Row) tea.Cmd {
			num, err := strconv.Atoi(row[0])
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: fmt.Errorf("invalid PR number: %s", row[0])}
				}
			}
			pr, err := client.GetPull(ctx, owner, repo, num)
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: err}
				}
			}
			title := fmt.Sprintf("#%d %s (%s → %s)", pr.Number, pr.Title, pr.Head.Ref, pr.Base.Ref)
			body := pr.Body
			if body == "" {
				body = "_No description provided._"
			}
			return tui.NewPagerCmd(title, body, tui.ContentMarkdown)
		},
		OnDiff: func(row table.Row) tea.Cmd {
			num, err := strconv.Atoi(row[0])
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: fmt.Errorf("invalid PR number: %s", row[0])}
				}
			}
			files, err := client.GetPullDiffFiles(ctx, owner, repo, num)
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: err}
				}
			}
			return tui.NewPagerCmd(fmt.Sprintf("PR #%s diff", row[0]), formatDiffFiles(files), tui.ContentDiff)
		},
		OnFetch: func(row table.Row) tea.Cmd {
			num, err := strconv.Atoi(row[0])
			if err != nil {
				return func() tea.Msg {
					return tui.FetchResultMsg{Err: fmt.Errorf("invalid PR number: %s", row[0])}
				}
			}
			localBranch := fmt.Sprintf("pr_%d", num)
			return func() tea.Msg {
				remotes, err := gitpkg.GiteeRemotes()
				if err != nil {
					return tui.FetchResultMsg{Err: err}
				}
				if err := gitpkg.FetchPR(remotes[0].URL, num, localBranch); err != nil {
					return tui.FetchResultMsg{Err: err}
				}
				return tui.FetchResultMsg{Branch: localBranch}
			}
		},
	})
}
