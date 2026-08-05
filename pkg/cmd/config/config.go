package config

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	cfg "gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func NewConfigCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Gitee CLI configuration",
		Long: `Manage Gitee CLI configuration.

Available configuration keys:

  DISPLAY
    tui              Enable interactive TUI mode (true/false, default: false)
    colorize         Colorize TUI table rows by state (true/false, default: false)
    theme            UI theme: auto, dark, light, dracula, tokyo-night, pink (TUI: interactive select)
    locale           UI language: en, zh_CN (default: auto-detected from $LANG; TUI: interactive select)

  EDITOR & PAGER
    editor           Editor for text input (priority: $GIT_EDITOR > $VISUAL > $EDITOR > this value > vim)
    pager            Pager for long output (e.g. less, more; default: system pager)

  CONNECTION
    host             Default Gitee hostname (default: gitee.com)
    api_prefix       API base URL (default: https://gitee.com/api/v5)
    api_swagger_url  OpenAPI spec URL (default: https://gitee.com/api/v5/swagger_doc.json)

  UPDATES
    update_check     Check for new releases at most once every 24 hours (true/false, default: true)

  REPOSITORY
    default_repo     Default owner/repo used outside a git repository (e.g. owner/repo)

  AI
    ai.base_url      OpenAI-compatible API base URL (e.g. https://api.openai.com/v1)
    ai.model         Model name (default: gpt-4o-mini)
    ai.token         API token ($GITEE_AI_TOKEN, or $OPENAI_API_KEY for api.openai.com)
    ai.language      Language for AI-generated content (default: English)

Examples:
  gitee config set tui true
  gitee config set colorize true
  gitee config set theme dark
  gitee config set locale zh_CN
  gitee config set theme        # TUI mode: interactive dropdown
  gitee config set locale       # TUI mode: interactive dropdown
  gitee config set editor nano
  gitee config set pager "less -R"
  gitee config set host gitee.com
  gitee config set api_prefix https://my-gitee.example.com/api/v5
  gitee config set update_check false
  gitee config set default_repo myorg/myrepo
  gitee config set ai.base_url https://api.openai.com/v1
  gitee config set ai.model gpt-4o
  gitee config set ai.token               # hidden prompt
  printf '%s' "$AI_TOKEN" | gitee config set ai.token --stdin
  gitee config set ai.language Chinese
  gitee config get tui
  gitee config list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newConfigGetCmd(f))
	cmd.AddCommand(newConfigSetCmd(f))
	cmd.AddCommand(newConfigListCmd(f))
	return cmd
}

func newConfigGetCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  `Get the value of a configuration key. Sensitive values are shown as "<redacted>".`,
		Example: `  gitee config get tui
  gitee config get host`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val := cfg.DisplayValue(args[0])
			fmt.Fprintln(f.IOStreams.Out, val)
			return nil
		},
	}
}

func newConfigSetCmd(f *cmdutil.Factory) *cobra.Command {
	const maxSensitiveValueBytes = 64 * 1024
	var fromStdin bool
	cmd := &cobra.Command{
		Use:   "set <key> [value]",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Sensitive values are read from a hidden prompt when the value is omitted. In scripts,
pass them through standard input with --stdin; command-line values are rejected.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])
			var value string

			if cfg.IsSensitiveKey(key) {
				if len(args) == 2 {
					return fmt.Errorf("do not pass %s on the command line; omit the value for a hidden prompt or use --stdin", key)
				}
				if fromStdin {
					data, err := io.ReadAll(io.LimitReader(f.IOStreams.In, maxSensitiveValueBytes+1))
					if err != nil {
						return fmt.Errorf("read %s from stdin: %w", key, err)
					}
					if len(data) > maxSensitiveValueBytes {
						return fmt.Errorf("%s exceeds the maximum supported length", key)
					}
					value = strings.TrimSpace(string(data))
				} else {
					if !f.IOStreams.IsStdinTerminal() {
						return fmt.Errorf("%s requires a terminal for hidden input; use --stdin in scripts", key)
					}
					var err error
					value, err = cmdutil.AskPassword(fmt.Sprintf("Enter %s", key), f.IsTUI())
					if err != nil {
						return err
					}
				}
				if value == "" {
					return fmt.Errorf("%s cannot be empty", key)
				}
			} else if fromStdin {
				return fmt.Errorf("--stdin is only supported for sensitive configuration values")
			} else if len(args) == 2 {
				value = args[1]
			} else if options, ok := cfg.ConfigOptions[key]; ok && f.IsTUI() {
				opts := make([]huh.Option[string], len(options))
				for i, o := range options {
					opts[i] = huh.NewOption(o, o)
				}
				if err := huh.NewForm(huh.NewGroup(
					huh.NewSelect[string]().
						Title(fmt.Sprintf("Select %s", key)).
						Options(opts...).
						Value(&value),
				)).Run(); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("value required: gitee config set %s <value>", key)
			}

			if err := cfg.Set(key, value); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "%s = %s\n", key, cfg.DisplayValue(key))
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read a sensitive value from standard input")
	return cmd
}

func flattenSettings(m map[string]interface{}, prefix string, out map[string]string) {
	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		if nested, ok := v.(map[string]interface{}); ok {
			flattenSettings(nested, fullKey, out)
		} else {
			out[fullKey] = fmt.Sprintf("%v", v)
		}
	}
}

func newConfigListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Long:  `List all configuration values. Sensitive values are shown as "<redacted>".`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings := cfg.AllSettings()
			if jsonOut {
				enc := json.NewEncoder(f.IOStreams.Out)
				return enc.Encode(settings)
			}

			flat := make(map[string]string, len(settings))
			flattenSettings(settings, "", flat)

			keys := make([]string, 0, len(flat))
			for k := range flat {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			if f.IsTUI() {
				columns := []table.Column{
					{Title: "Key", Width: 24},
					{Title: "Value", Width: 50},
				}
				rows := make([]table.Row, 0, len(keys))
				for _, k := range keys {
					rows = append(rows, table.Row{k, flat[k]})
				}
				return tui.RunTable(tui.TableConfig{
					Columns: columns,
					Rows:    rows,
					Height:  min(len(rows)+1, 20),
					HelpKeys: []tui.HelpKey{
						{Key: "q", Desc: "quit"},
					},
				})
			}

			for _, k := range keys {
				fmt.Fprintf(f.IOStreams.Out, "%s=%s\n", k, flat[k])
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
