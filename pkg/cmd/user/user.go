package user

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func NewUserCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	cmd.AddCommand(newUserSearchCmd(f))
	return cmd
}

func newUserSearchCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for users",
		Long:  `Search for Gitee users by keyword. Searches login names, display names, and other profile fields.`,
		Example: `  gitee user search alice
  gitee user search "alice bob"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			users, err := client.SearchUsers(f.Context, args[0])
			if err != nil {
				return fmt.Errorf("failed to search users: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.User](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, users)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, users, fields)
			}

			if len(users) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No users found")
				return nil
			}

			if f.IsTUI() {
				return userSearchTUI(f.Context, users, client)
			}

			rows := [][]string{{"LOGIN", "NAME", "URL"}}
			for _, u := range users {
				rows = append(rows, []string{u.Login, u.Name, u.HTMLURL})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.User]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func userSearchTUI(ctx context.Context, users []gitee.User, client *gitee.Client) error {
	columns := []table.Column{
		{Title: "Login", Width: 20},
		{Title: "Name", Width: 20},
		{Title: "URL", Width: 40},
	}

	rows := make([]table.Row, 0, len(users))
	for _, u := range users {
		rows = append(rows, table.Row{u.Login, u.Name, u.HTMLURL})
	}

	return tui.RunTable(tui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Height:  min(len(users)+1, 15),
		HelpKeys: []tui.HelpKey{
			{Key: "enter", Desc: "open"},
			{Key: "v", Desc: "preview"},
			{Key: "c", Desc: "copy login"},
			{Key: "q", Desc: "quit"},
		},
		OnSelect: func(row table.Row) {
			browser.OpenURL(row[2])
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(row[0])
		},
		OnView: func(row table.Row) tea.Cmd {
			u, err := client.GetUser(ctx, row[0])
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: err}
				}
			}
			content := fmt.Sprintf("Login:  %s\nName:   %s\nURL:    %s\nType:   %s",
				u.Login, u.Name, u.HTMLURL, u.Type)
			return tui.NewPagerCmd(u.Login, content, tui.ContentMarkdown)
		},
	})
}
