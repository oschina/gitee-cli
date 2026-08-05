package repo

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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

func NewRepoCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
		Long:  `List, view, clone, create, fork, and delete repositories.`,
	}

	cmd.AddCommand(newRepoListCmd(f))
	cmd.AddCommand(newRepoViewCmd(f))
	cmd.AddCommand(newRepoCloneCmd(f))
	cmd.AddCommand(newRepoCreateCmd(f))
	cmd.AddCommand(newRepoForkCmd(f))
	cmd.AddCommand(newRepoDeleteCmd(f))
	return cmd
}

func newRepoListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit       int
		page        int
		sort        string
		repoType    string
		affiliation string
		search      string
		jsonFields  string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List your repositories",
		Long:    `List repositories you own or have access to. Supports filtering by type, affiliation, and search keyword.`,
		Aliases: []string{"ls"},
		Example: `  gitee repo list
  gitee repo list --type owner
  gitee repo list --search gitee-cli
  gitee repo list --json=full_name,description,language,stargazers_count`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			params := &gitee.ListReposParams{
				Sort:        sort,
				Type:        repoType,
				Affiliation: affiliation,
				Q:           search,
				Page:        page,
				PerPage:     limit,
			}

			repos, err := client.ListUserRepos(f.Context, params)
			if err != nil {
				return fmt.Errorf("failed to list repos: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Repository](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, repos)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, repos, fields)
			}

			if len(repos) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("repo.no_results"))
				return nil
			}

			if f.IsTUI() {
				return repoListTUI(f.Context, repos, f.Hostname, client)
			}

			rows := [][]string{{"NAME", "DESCRIPTION", "STARS", "LANG", "VISIBILITY"}}
			for _, r := range repos {
				vis := "public"
				if r.Private {
					vis = "private"
				} else if r.Internal {
					vis = "internal"
				}
				desc := tui.Truncate(r.Description, 40)
				lang := r.Language
				if lang == "" {
					lang = "-"
				}
				rows = append(rows, []string{
					r.FullName,
					desc,
					strconv.Itoa(r.StargazersCount),
					lang,
					vis,
				})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of repos per page (max 100)")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().StringVarP(&sort, "sort", "s", "full_name", "Sort by: created, updated, pushed, full_name")
	cmd.Flags().StringVar(&repoType, "type", "", "Filter by: owner, personal, member, public, private")
	cmd.Flags().StringVar(&affiliation, "affiliation", "", "Filter by: owner, collaborator, organization_member")
	cmd.Flags().StringVar(&search, "search", "", "Search keyword")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Repository]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func repoListTUI(ctx context.Context, repos []gitee.Repository, hostname string, client *gitee.Client) error {
	columns := []table.Column{
		{Title: "Name", Width: 35},
		{Title: "Description", Width: 30},
		{Title: "★", Width: 5},
		{Title: "Lang", Width: 10},
		{Title: "Vis", Width: 8},
	}

	rows := make([]table.Row, 0, len(repos))
	for _, r := range repos {
		vis := "public"
		if r.Private {
			vis = "private"
		} else if r.Internal {
			vis = "internal"
		}
		desc := tui.Truncate(r.Description, 28)
		lang := r.Language
		if lang == "" {
			lang = "-"
		}
		rows = append(rows, table.Row{r.FullName, desc, strconv.Itoa(r.StargazersCount), lang, vis})
	}

	return tui.RunTable(tui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Height:  min(len(repos)+1, 20),
		HelpKeys: []tui.HelpKey{
			{Key: "enter", Desc: "open"},
			{Key: "v", Desc: "preview"},
			{Key: "c", Desc: "copy"},
			{Key: "q", Desc: "quit"},
		},
		OnSelect: func(row table.Row) {
			if hostname == "" {
				hostname = config.DefaultHost
			}
			url := fmt.Sprintf("https://%s/%s", hostname, row[0])
			browser.OpenURL(url)
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(row[0])
		},
		OnView: func(row table.Row) tea.Cmd {
			parts := strings.SplitN(row[0], "/", 2)
			if len(parts) != 2 {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: fmt.Errorf("invalid repo name: %s", row[0])}
				}
			}
			r, err := client.GetRepo(ctx, parts[0], parts[1])
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: err}
				}
			}
			title := fmt.Sprintf("%s — %s", r.FullName, visibilityLabel(r))
			return tui.NewPagerCmd(title, repoMarkdown(r), tui.ContentMarkdown)
		},
	})
}
