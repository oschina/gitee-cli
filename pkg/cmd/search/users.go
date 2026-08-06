package search

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newUsersCmd(f *cmdutil.Factory) *cobra.Command {
	var opts commonOptions
	cmd := &cobra.Command{
		Use:     "users <query>",
		Aliases: []string{"user"},
		Short:   "Search users",
		Example: `  gitee search users alice
  gitee search users "alice bob" --sort joined_at`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.validate([]string{"joined_at"})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			users, err := client.SearchUsersWithParams(f.Context, &gitee.SearchUsersParams{
				Query: args[0], Sort: opts.sort, Order: opts.order, Page: opts.page, PerPage: opts.limit,
			})
			if err != nil {
				return fmt.Errorf("failed to search users: %w", err)
			}
			if opts.jsonFields != "" {
				return writeJSON[gitee.User](f, opts.jsonFields, users)
			}
			if len(users) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("user.no_results"))
				return nil
			}
			if f.IsTUI() {
				return usersTUI(f.Context, users, client)
			}
			rows := [][]string{{"LOGIN", "NAME", "URL"}}
			for _, user := range users {
				rows = append(rows, []string{user.Login, user.Name, user.HTMLURL})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	opts.addFlags(cmd, "users", "joined_at")
	cmd.Flags().StringVarP(&opts.jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.User]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func usersTUI(ctx context.Context, users []gitee.User, client *gitee.Client) error {
	columns := []table.Column{{Title: "Login", Width: 20}, {Title: "Name", Width: 20}, {Title: "URL", Width: 40}}
	rows := make([]table.Row, 0, len(users))
	for _, user := range users {
		rows = append(rows, table.Row{user.Login, user.Name, user.HTMLURL})
	}
	return tui.RunTable(tui.TableConfig{
		Columns: columns, Rows: rows, Height: min(len(users)+1, 15),
		HelpKeys: []tui.HelpKey{{Key: "enter", Desc: "open"}, {Key: "v", Desc: "preview"}, {Key: "c", Desc: "copy login"}, {Key: "q", Desc: "quit"}},
		OnSelect: func(row table.Row) { browser.OpenURL(row[2]) },
		OnCopy:   func(row table.Row) { cmdutil.CopyToClipboard(row[0]) },
		OnView: func(row table.Row) tea.Cmd {
			user, err := client.GetUser(ctx, row[0])
			if err != nil {
				return func() tea.Msg { return tui.ViewErrorMsg{Err: err} }
			}
			content := fmt.Sprintf("Login:  %s\nName:   %s\nURL:    %s\nType:   %s", user.Login, user.Name, user.HTMLURL, user.Type)
			return tui.NewPagerCmd(user.Login, content, tui.ContentMarkdown)
		},
	})
}
