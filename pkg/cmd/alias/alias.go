package alias

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

const aliasPrefix = "alias."

func NewAliasCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Manage command aliases",
		Long: `Manage command aliases.

Aliases expand into full commands. Positional variables $1, $2, ... are
replaced by the arguments passed after the alias name, in order. If no
positional variables appear in the expansion, extra arguments are appended
to the end.

Prefix the expansion with ! to run an arbitrary shell command instead of a
gitee sub-command. Positional variables and extra-argument appending work
the same way. The command is executed via sh -c on Unix or cmd /c on Windows.

Examples:
  gitee alias set prs "pr list -s open"
  gitee prs                     → gitee pr list -s open

  gitee alias set build "pr comment $1 -b $2"
  gitee build 123 deployed      → gitee pr comment 123 -b deployed

  gitee alias set myissues "issue list -A $1 -s open"
  gitee myissues alice           → gitee issue list -A alice -s open

  gitee alias set deploy "!PR=$(gitee pr create --base $1 --json | jq -r .number) && gitee pr comment $PR -b ci_deploy"
  gitee deploy main              → creates a PR against main, then comments ci_deploy`,
	}
	cmd.AddCommand(newAliasListCmd(f))
	cmd.AddCommand(newAliasSetCmd(f))
	cmd.AddCommand(newAliasDeleteCmd(f))
	return cmd
}

func newAliasListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List aliases",
		Long:    `List all configured command aliases and their expansions.`,
		Aliases: []string{"ls"},
		Example: `  gitee alias list`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			aliases := collectAliases()

			if jsonOut {
				return json.NewEncoder(f.IOStreams.Out).Encode(aliases)
			}
			if len(aliases) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("alias.no_results"))
				return nil
			}

			names := sortedNames(aliases)

			if f.IsTUI() {
				return runAliasTUI(names, aliases)
			}

			w := tabwriter.NewWriter(f.IOStreams.Out, 0, 0, 2, ' ', 0)
			for _, name := range names {
				fmt.Fprintf(w, "%s\t%s\n", name, aliases[name])
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func collectAliases() map[string]string {
	all := config.AllSettings()
	aliases := map[string]string{}
	if aliasMap, ok := all["alias"]; ok {
		if m, ok := aliasMap.(map[string]interface{}); ok {
			for k, v := range m {
				if s := fmt.Sprintf("%v", v); s != "" {
					aliases[k] = s
				}
			}
		}
	}
	for k, v := range all {
		if len(k) > len(aliasPrefix) && k[:len(aliasPrefix)] == aliasPrefix {
			name := k[len(aliasPrefix):]
			if s := fmt.Sprintf("%v", v); s != "" {
				aliases[name] = s
			}
		}
	}
	return aliases
}

func sortedNames(aliases map[string]string) []string {
	names := make([]string, 0, len(aliases))
	for k := range aliases {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func runAliasTUI(names []string, aliases map[string]string) error {
	rows := make([]table.Row, len(names))
	for i, name := range names {
		rows[i] = table.Row{name, aliases[name]}
	}

	return tui.RunTable(tui.TableConfig{
		Columns: []table.Column{
			{Title: "Alias", Width: 20},
			{Title: "Expansion", Width: 60},
		},
		Rows:   rows,
		Height: min(len(rows)+1, 20),
		HelpKeys: []tui.HelpKey{
			{Key: "v", Desc: "view"},
			{Key: "e", Desc: "edit"},
			{Key: "c", Desc: "copy expansion"},
			{Key: "q", Desc: "quit"},
		},
		OnView: func(row table.Row) tea.Cmd {
			name := row[0]
			expansion := aliases[name]
			return tui.NewPagerCmd(name, expansion, tui.ContentPlain)
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(aliases[row[0]])
		},
		OnEdit: func(row table.Row) tea.ExecCommand {
			name := row[0]
			return tui.NewHuhExecCmd(func() error {
				expansion := aliases[name]
				editorParts := strings.Fields(config.Editor())
				if err := huh.NewForm(huh.NewGroup(
					huh.NewText().
						Title(fmt.Sprintf("Edit alias: %s", name)).
						Editor(editorParts...).
						Value(&expansion),
				)).Run(); err != nil {
					return err
				}
				if expansion == "" {
					return fmt.Errorf("expansion cannot be empty")
				}
				if err := config.Set(aliasPrefix+name, expansion); err != nil {
					return err
				}
				aliases[name] = expansion
				return nil
			})
		},
	})
}

func newAliasSetCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "set <alias> <expansion>",
		Short: "Set an alias",
		Long: `Set a command alias. The alias expands into a full command.

Positional variables $1, $2, ... are replaced by arguments passed
after the alias name. If no positional variables appear, extra
arguments are appended to the end.

Prefix the expansion with ! to run an arbitrary shell command
instead of a gitee sub-command.`,
		Example: `  gitee alias set prs "pr list -s open"
  gitee alias set myissues "issue list -A $1 -s open"
  gitee alias set deploy "!PR=$(gitee pr create --base $1 --json | jq -r .number) && gitee pr comment $PR -b ci_deploy"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, expansion := args[0], args[1]
			if err := config.Set(aliasPrefix+name, expansion); err != nil {
				return fmt.Errorf("failed to save alias: %w", err)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("alias.added", name, expansion))
			return nil
		},
	}
}

func newAliasDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <alias>",
		Short:   "Delete an alias",
		Long:    `Delete a previously configured command alias by name.`,
		Aliases: []string{"rm"},
		Example: `  gitee alias delete prs
  gitee alias delete myissues`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			key := aliasPrefix + name
			if config.Get(key) == "" {
				return fmt.Errorf("alias %q not found", name)
			}
			if err := config.Set(key, ""); err != nil {
				return fmt.Errorf("failed to delete alias: %w", err)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("alias.deleted", name))
			return nil
		},
	}
}
