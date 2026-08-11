package issue

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newIssueListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		state      string
		labels     string
		query      string
		assignee   string
		sort       string
		direction  string
		limit      int
		page       int
		jsonFields string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List issues",
		Long:    `List issues in the repository. Supports filtering by state, labels, assignee, and keyword search.`,
		Aliases: []string{"ls"},
		Example: `  gitee issue list
  gitee issue list -s open
  gitee issue list --labels bug,urgent -A alice
  gitee issue list --sort updated --direction desc
  gitee issue list --json=number,title,state
  gitee issue list --json=fields`,
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

			issues, err := client.ListRepoIssues(f.Context, owner, repo, &gitee.ListIssuesParams{
				State:     state,
				Labels:    labels,
				Q:         query,
				Assignee:  assignee,
				Sort:      sort,
				Direction: direction,
				Page:      page,
				PerPage:   limit,
			})
			if err != nil {
				return fmt.Errorf("failed to list issues: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Issue](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, issues)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, issues, fields)
			}

			if len(issues) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("issue.no_results"))
				return nil
			}

			if f.IsTUI() {
				return issueListTUI(f.Context, issues, owner, repo, f.Hostname, client)
			}

			rows := [][]string{{"#", "TITLE", "STATE", "ASSIGNEE", "LABELS", "CREATOR"}}
			for _, iss := range issues {
				rows = append(rows, []string{
					iss.Number,
					tui.Truncate(iss.Title, 40),
					iss.State,
					assigneeLogin(iss),
					labelNames(iss),
					iss.User.Login,
				})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	cmd.Flags().StringVarP(&state, "state", "s", "open", "Filter by state: open, progressing, closed, rejected, all")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&query, "query", "", "Search keyword")
	cmd.Flags().StringVarP(&assignee, "assignee", "A", "", "Filter by assignee username")
	cmd.Flags().StringVar(&sort, "sort", "created", "Sort by: created, updated")
	cmd.Flags().StringVar(&direction, "direction", "desc", "Sort direction: asc, desc")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of issues per page")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Issue]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func assigneeLogin(iss gitee.Issue) string {
	if iss.Assignee != nil {
		return iss.Assignee.Login
	}
	return "-"
}

func labelNames(iss gitee.Issue) string {
	names := issueLabelNames(iss.Labels)
	if names == "" {
		return "-"
	}
	return names
}

func issueListTUI(ctx context.Context, issues []gitee.Issue, owner, repo, hostname string, client *gitee.Client) error {
	columns := []table.Column{
		{Title: "#", Width: 10},
		{Title: "Title", Width: 35},
		{Title: "State", Width: 12},
		{Title: "Assignee", Width: 14},
		{Title: "Labels", Width: 16},
		{Title: "Creator", Width: 12},
	}

	rows := make([]table.Row, 0, len(issues))
	for _, iss := range issues {
		rows = append(rows, table.Row{
			iss.Number,
			tui.Truncate(iss.Title, 33),
			iss.State,
			assigneeLogin(iss),
			tui.Truncate(labelNames(iss), 14),
			iss.User.Login,
		})
	}

	return tui.RunTable(tui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Height:  min(len(issues)+1, 20),
		HelpKeys: []tui.HelpKey{
			{Key: "enter", Desc: "open"},
			{Key: "v", Desc: "preview"},
			{Key: "c", Desc: "copy ident"},
			{Key: "e", Desc: "edit"},
			{Key: "q", Desc: "quit"},
		},
		OnSelect: func(row table.Row) {
			if hostname == "" {
				hostname = config.DefaultHost
			}
			url := fmt.Sprintf("https://%s/%s/%s/issues/%s", hostname, owner, repo, row[0])
			browser.OpenURL(url)
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(row[0])
		},
		OnView: func(row table.Row) tea.Cmd {
			iss, err := client.GetIssue(ctx, owner, repo, row[0])
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: err}
				}
			}
			title := fmt.Sprintf("#%s %s", iss.Number, iss.Title)
			body := iss.Body
			if body == "" {
				body = "_No description provided._"
			}
			return tui.NewPagerCmd(title, body, tui.ContentMarkdown)
		},
		OnEdit: func(row table.Row) tea.ExecCommand {
			number := row[0]
			return tui.NewHuhExecCmd(func() error {
				iss, err := client.GetIssue(ctx, owner, repo, number)
				if err != nil {
					return err
				}
				title := iss.Title
				body := iss.Body
				assignee := ""
				if iss.Assignee != nil {
					assignee = iss.Assignee.Login
				}
				labels := issueLabelNames(iss.Labels)
				if err := issueEditForm(&title, &body, &assignee, &labels); err != nil {
					return err
				}
				params := &gitee.UpdateIssueParams{
					Title:    title,
					Body:     &body,
					Assignee: &assignee,
					Labels:   &labels,
				}
				_, err = client.UpdateIssue(ctx, owner, repo, number, params)
				return err
			})
		},
	})
}
