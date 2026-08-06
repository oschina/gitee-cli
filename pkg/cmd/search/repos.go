package search

import (
	"context"
	"fmt"
	"strconv"
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

func newReposCmd(f *cmdutil.Factory) *cobra.Command {
	var opts commonOptions
	var owner, language string
	var includeForks bool

	cmd := &cobra.Command{
		Use:     "repos <query>",
		Aliases: []string{"repo", "repositories"},
		Short:   "Search repositories",
		Example: `  gitee search repos gitee-cli
  gitee search repos sdk --owner oschina --language Go
  gitee search repos database --sort stars_count --order desc`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.validate([]string{"last_push_at", "stars_count", "forks_count", "watches_count"})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			repos, err := client.SearchRepositories(f.Context, &gitee.SearchRepositoriesParams{
				Query: args[0], Owner: owner, Fork: includeForks, Language: language,
				Sort: opts.sort, Order: opts.order, Page: opts.page, PerPage: opts.limit,
			})
			if err != nil {
				return fmt.Errorf("failed to search repositories: %w", err)
			}
			if opts.jsonFields != "" {
				return writeJSON[gitee.Repository](f, opts.jsonFields, repos)
			}
			if len(repos) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("repo.no_results"))
				return nil
			}
			if f.IsTUI() {
				return reposTUI(f.Context, repos, client)
			}
			rows := [][]string{{"NAME", "DESCRIPTION", "STARS", "LANG", "UPDATED"}}
			for _, repo := range repos {
				language := repo.Language
				if language == "" {
					language = "-"
				}
				rows = append(rows, []string{repo.FullName, tui.Truncate(repo.Description, 40), strconv.Itoa(repo.StargazersCount), language, formatDate(repo.UpdatedAt)})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	opts.addFlags(cmd, "repositories", "last_push_at, stars_count, forks_count, watches_count")
	cmd.Flags().StringVar(&owner, "owner", "", "Filter by namespace path")
	cmd.Flags().BoolVar(&includeForks, "fork", false, "Include forked repositories")
	cmd.Flags().StringVar(&language, "language", "", "Filter by programming language")
	cmd.Flags().StringVarP(&opts.jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Repository]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func reposTUI(ctx context.Context, repos []gitee.Repository, client *gitee.Client) error {
	columns := []table.Column{{Title: "Name", Width: 35}, {Title: "Description", Width: 30}, {Title: "Stars", Width: 7}, {Title: "Lang", Width: 12}}
	rows := make([]table.Row, 0, len(repos))
	urls := make(map[string]string, len(repos))
	for _, repo := range repos {
		language := repo.Language
		if language == "" {
			language = "-"
		}
		rows = append(rows, table.Row{repo.FullName, tui.Truncate(repo.Description, 28), strconv.Itoa(repo.StargazersCount), language})
		urls[repo.FullName] = repo.HTMLURL
	}
	return tui.RunTable(tui.TableConfig{
		Columns: columns, Rows: rows, Height: min(len(repos)+1, 20),
		HelpKeys: []tui.HelpKey{{Key: "enter", Desc: "open"}, {Key: "v", Desc: "preview"}, {Key: "c", Desc: "copy"}, {Key: "q", Desc: "quit"}},
		OnSelect: func(row table.Row) { browser.OpenURL(urls[row[0]]) },
		OnCopy:   func(row table.Row) { cmdutil.CopyToClipboard(row[0]) },
		OnView: func(row table.Row) tea.Cmd {
			parts := strings.SplitN(row[0], "/", 2)
			if len(parts) != 2 {
				return func() tea.Msg { return tui.ViewErrorMsg{Err: fmt.Errorf("invalid repository name: %s", row[0])} }
			}
			repo, err := client.GetRepo(ctx, parts[0], parts[1])
			if err != nil {
				return func() tea.Msg { return tui.ViewErrorMsg{Err: err} }
			}
			body := repo.Description
			if body == "" {
				body = "_No description provided._"
			}
			return tui.NewPagerCmd(repo.FullName, body, tui.ContentMarkdown)
		},
	})
}
