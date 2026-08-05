package sshkey

import (
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

func keyComment(key string) string {
	parts := strings.Fields(key)
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return "-"
}

func keyPreview(key string) string {
	parts := strings.Fields(key)
	if len(parts) >= 2 {
		return parts[0] + " " + tui.Truncate(parts[1], 30)
	}
	return tui.Truncate(key, 36)
}

func newSSHKeyListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List your SSH keys",
		Long:    `List all SSH public keys registered on your Gitee account.`,
		Aliases: []string{"ls"},
		Example: `  gitee ssh-key list
  gitee ssh-key list --json=id,key`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			user, err := client.GetAuthenticatedUser(f.Context)
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			keys, err := client.ListSSHKeys(f.Context, user.Login)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.SSHKey](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, keys)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, keys, fields)
			}

			if len(keys) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("sshkey.no_results"))
				return nil
			}

			if f.IsTUI() {
				return sshKeyListTUI(keys, f.Hostname)
			}

			rows := [][]string{{"ID", "COMMENT", "KEY"}}
			for _, k := range keys {
				rows = append(rows, []string{
					strconv.Itoa(k.ID),
					keyComment(k.Key),
					keyPreview(k.Key),
				})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.SSHKey]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func sshKeyListTUI(keys []gitee.SSHKey, hostname string) error {
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Comment", Width: 25},
		{Title: "Key Preview", Width: 45},
		{Title: "", Width: 0},
	}

	rows := make([]table.Row, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", k.ID),
			keyComment(k.Key),
			keyPreview(k.Key),
			k.Key,
		})
	}

	return tui.RunTable(tui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Height:  min(len(keys)+1, 15),
		HelpKeys: []tui.HelpKey{
			{Key: "enter", Desc: "open"},
			{Key: "v", Desc: "preview"},
			{Key: "c", Desc: "copy key"},
			{Key: "q", Desc: "quit"},
		},
		OnSelect: func(row table.Row) {
			if hostname == "" {
				hostname = config.DefaultHost
			}
			browser.OpenURL(fmt.Sprintf("https://%s/keys/%s", hostname, row[0]))
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(row[3])
		},
		OnView: func(row table.Row) tea.Cmd {
			title := fmt.Sprintf("SSH Key #%s", row[0])
			content := fmt.Sprintf("```\n%s\n```", row[3])
			return tui.NewPagerCmd(title, content, tui.ContentMarkdown)
		},
	})
}
