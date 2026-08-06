package search

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newIssuesCmd(f *cmdutil.Factory) *cobra.Command {
	var opts commonOptions
	var repo, language, label, state, author, assignee string

	cmd := &cobra.Command{
		Use:     "issues <query>",
		Aliases: []string{"issue"},
		Short:   "Search issues",
		Example: `  gitee search issues timeout
  gitee search issues bug --repo oschina/gitee --state open
  gitee search issues crash --label bug --assignee alice`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.validate([]string{"created_at", "updated_at", "notes_count"}); err != nil {
				return err
			}
			if state != "" && !slices.Contains([]string{"open", "progressing", "closed", "rejected"}, state) {
				return cmdutil.FlagErrorf("--state must be one of: open, progressing, closed, rejected")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			issues, err := client.SearchIssues(f.Context, &gitee.SearchIssuesParams{
				Query: args[0], Repo: repo, Language: language, Label: label, State: state,
				Author: author, Assignee: assignee, Sort: opts.sort, Order: opts.order, Page: opts.page, PerPage: opts.limit,
			})
			if err != nil {
				return fmt.Errorf("failed to search issues: %w", err)
			}
			if opts.jsonFields != "" {
				return writeJSON[gitee.Issue](f, opts.jsonFields, issues)
			}
			if len(issues) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("issue.no_results"))
				return nil
			}
			if f.IsTUI() {
				return issuesTUI(issues)
			}
			rows := [][]string{{"REPOSITORY", "#", "TITLE", "STATE", "CREATOR", "UPDATED"}}
			for _, issue := range issues {
				rows = append(rows, []string{issue.Repository.FullName, issue.Number, tui.Truncate(issue.Title, 40), issue.State, issue.User.Login, formatDate(issue.UpdatedAt)})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	opts.addFlags(cmd, "issues", "created_at, updated_at, notes_count")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "Filter by repository (OWNER/REPO)")
	cmd.Flags().StringVar(&language, "language", "", "Filter by repository language")
	cmd.Flags().StringVar(&label, "label", "", "Filter by label")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Filter by state: open, progressing, closed, rejected")
	cmd.Flags().StringVar(&author, "author", "", "Filter by author username")
	cmd.Flags().StringVarP(&assignee, "assignee", "A", "", "Filter by assignee username")
	cmd.Flags().StringVarP(&opts.jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Issue]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func issuesTUI(issues []gitee.Issue) error {
	columns := []table.Column{{Title: "Repository", Width: 28}, {Title: "#", Width: 10}, {Title: "Title", Width: 36}, {Title: "State", Width: 12}}
	rows := make([]table.Row, 0, len(issues))
	byKey := make(map[string]gitee.Issue, len(issues))
	for _, issue := range issues {
		rows = append(rows, table.Row{issue.Repository.FullName, issue.Number, tui.Truncate(issue.Title, 34), issue.State})
		byKey[issue.Repository.FullName+"\x00"+issue.Number] = issue
	}
	find := func(row table.Row) gitee.Issue { return byKey[row[0]+"\x00"+row[1]] }
	return tui.RunTable(tui.TableConfig{
		Columns: columns, Rows: rows, Height: min(len(issues)+1, 20),
		HelpKeys: []tui.HelpKey{{Key: "enter", Desc: "open"}, {Key: "v", Desc: "preview"}, {Key: "c", Desc: "copy ident"}, {Key: "q", Desc: "quit"}},
		OnSelect: func(row table.Row) { browser.OpenURL(find(row).HTMLURL) },
		OnCopy:   func(row table.Row) { cmdutil.CopyToClipboard(row[1]) },
		OnView: func(row table.Row) tea.Cmd {
			issue := find(row)
			body := issue.Body
			if strings.TrimSpace(body) == "" {
				body = "_No description provided._"
			}
			return tui.NewPagerCmd(fmt.Sprintf("%s #%s: %s", issue.Repository.FullName, issue.Number, issue.Title), body, tui.ContentMarkdown)
		},
	})
}
