package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// NewPipelinePluginCmd returns the pipeline plugin command group.
func newPipelinePluginCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage gitee-go plugins",
		Long:  `List gitee-go plugins and inspect their parameter schemes / example YAML.`,
	}
	cmd.AddCommand(newPipelinePluginListCmd(f))
	cmd.AddCommand(newPipelinePluginSchemeCmd(f))
	cmd.AddCommand(newPipelinePluginExampleCmd(f))
	return cmd
}

// resolvePluginClient resolves -R owner/repo plus the repo-scoped plugin client
// (with service-status check). Shared by the plugin subcommands.
func resolvePluginClient(f *cmdutil.Factory, cmd *cobra.Command) (*giteego.Client, error) {
	owner, repo, err := resolveOwnerRepo(f, cmd)
	if err != nil {
		return nil, err
	}
	client, err := repoPipelineClientForService(f, owner, repo, giteego.ServiceIPipe)
	if err != nil {
		return nil, err
	}
	if err := ensureServiceOpen(f, cmd, owner, repo); err != nil {
		return nil, err
	}
	return client, nil
}

func newPipelinePluginListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available gitee-go plugins",
		Long:  `List the gitee-go plugins available to the repository, grouped by category.`,
		Example: `  gitee pipeline plugin list -R owner/repo
  gitee pipeline plugin list -R owner/repo --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolvePluginClient(f, cmd)
			if err != nil {
				return err
			}
			plugins, err := client.ListPlugins(f.Context)
			if err != nil {
				return fmt.Errorf("failed to list plugins: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.CategoryPluginVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, plugins)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, plugins, fields)
			}

			// Table output, aligned with the gitee CLI conventions.
			rows := [][]string{{"CATEGORY", "NAME", "TYPE", "DESCRIPTION"}}
			for _, cat := range plugins {
				if len(cat.Plugins) == 0 {
					rows = append(rows, []string{cat.Name, "-", "-", "-"})
					continue
				}
				for i, p := range cat.Plugins {
					catName := ""
					if i == 0 {
						catName = cat.Name
					}
					rows = append(rows, []string{catName, p.Name, p.Type, p.Description})
				}
			}
			return cmdutil.WriteTableBordered(f.IOStreams.Out, rows)
		},
	}

	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.CategoryPluginVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelinePluginSchemeCmd(f *cmdutil.Factory) *cobra.Command {
	var jobType string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "scheme --type <jobType>",
		Short: "Show a plugin's parameter scheme",
		Long:  `Show the parameter scheme (fields, defaults, validation) of a gitee-go plugin for a job type. The --type value is the plugin's JSON type as shown by 'plugin list' (e.g. JENKINS_JOB).`,
		Example: `  gitee pipeline plugin scheme -R owner/repo --type JENKINS_JOB
  gitee pipeline plugin scheme -R owner/repo --type maven-build@v1.0.0 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolvePluginClient(f, cmd)
			if err != nil {
				return err
			}
			scheme, err := client.GetPluginScheme(f.Context, jobType)
			if err != nil {
				return fmt.Errorf("failed to get plugin scheme: %w", err)
			}
			if scheme == nil {
				return fmt.Errorf("no scheme for job type %q", jobType)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PluginSchemeVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, scheme)
				}
				result, err := cmdutil.SelectFields(scheme, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			out := f.IOStreams.Out
			// Header block: name, type pair, description — separated by blank lines.
			fmt.Fprintf(out, "%s\n", scheme.Name)
			fmt.Fprintf(out, "Type: %s   YAML: %s\n", scheme.Type.JSON, scheme.Type.YAML)
			if scheme.Description != "" {
				fmt.Fprintln(out)
				fmt.Fprintf(out, "%s\n", scheme.Description)
			}
			fmt.Fprintln(out)
			if scheme.Doc != "" {
				fmt.Fprintf(out, "Doc:  %s\n", scheme.Doc)
			}
			if scheme.Advance != nil {
				fmt.Fprintf(out, "Adv:  notification=%t  timeout=%t  skip=%t  retry=%t\n",
					scheme.Advance.Notification, scheme.Advance.Timeout, scheme.Advance.Skip, scheme.Advance.Retry)
			}
			if scheme.Source != nil {
				fmt.Fprintf(out, "Src:  enabled=%t  multiple=%t\n", scheme.Source.Enabled, scheme.Source.Multiple)
			}
			if scheme.Cache != nil && scheme.Cache.Enabled {
				fmt.Fprintf(out, "Cac:  enabled=%t  default=%s\n", scheme.Cache.Enabled, scheme.Cache.DefaultValue)
			}
			if len(scheme.OutputParams) > 0 {
				fmt.Fprintf(out, "Out:  %s\n", strings.Join(scheme.OutputParams, ", "))
			}
			if len(scheme.Config) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, i18n.T("pipeline.config_params"))
				fmt.Fprintln(out, "--------")
				printSchemeConfig(out, scheme.Config)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&jobType, "type", "", "Plugin JSON job type (from 'plugin list')")
	_ = cmd.MarkFlagRequired("type")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PluginSchemeVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// printSchemeConfig prints the scheme's config, one component per paragraph
// with a blank line between them. Complex default values render as compact JSON
// (no Go %v map noise); i18n keys (#{{...}}) and {{...}} refs are left as-is
// since they are placeholders resolved at runtime.
func printSchemeConfig(w io.Writer, config []giteego.PluginComponentScheme) {
	for i, c := range config {
		if isRefValue(c) {
			// RefValue components are template refs only useful for JSON pipeline
			// submission (e.g. {{CERTIFICATION.${certificate}.server}}); irrelevant
			// to YAML, so skip them.
			continue
		}
		if i > 0 {
			fmt.Fprintln(w)
		}
		name := i18nName(c.Name)
		if name == "" {
			name = c.Identifier
		}
		// First line: readable name + type + required/array markers.
		markers := componentMarkers(c)
		fmt.Fprintf(w, "  %s  [%s]%s\n", name, c.Type, markers)

		if c.Identifier != "" {
			fmt.Fprintf(w, "    id:      %s\n", c.Identifier)
		}
		if c.Convertor != nil && c.Convertor.YAMLField != "" {
			fmt.Fprintf(w, "    yaml:    %s\n", c.Convertor.YAMLField)
		}
		if c.Type == "RemoteSelect" {
			printRemoteURL(w, c)
		}
		if def := formatDefault(c.DefaultValue); def != "" {
			fmt.Fprintf(w, "    default: %s\n", def)
		}
		if req := formatRules(c); req != "" {
			fmt.Fprintf(w, "    rules:   %s\n", req)
		}

		// Compose children indented under the parent (RefValue children skipped).
		var children []giteego.PluginComponentScheme
		for _, ch := range c.Children {
			if !isRefValue(ch) {
				children = append(children, ch)
			}
		}
		if len(children) > 0 {
			fmt.Fprintln(w)
			for _, child := range children {
				cname := i18nName(child.Name)
				if cname == "" {
					cname = child.Identifier
				}
				line := fmt.Sprintf("      - %s  [%s]%s", cname, child.Type, componentMarkers(child))
				if cdef := formatDefault(child.DefaultValue); cdef != "" {
					line += fmt.Sprintf("  (default: %s)", cdef)
				}
				fmt.Fprintln(w, line)
			}
		}
	}
}

// isRefValue reports whether a scheme component is a RefValue (JSON-only
// template ref, not applicable to YAML).
func isRefValue(c giteego.PluginComponentScheme) bool {
	return c.Type == "RefValue"
}

// componentMarkers renders suffix markers for a component: (required) and
// (array) when applicable.
func componentMarkers(c giteego.PluginComponentScheme) string {
	markers := ""
	for _, r := range c.Rules {
		if r.Required {
			markers += "  (required)"
			break
		}
	}
	if c.Array {
		markers += "  (array)"
	}
	return markers
}

// printRemoteURL shows the remote-select endpoint path for a RemoteSelect
// component so callers know what `plugin request` will hit.
func printRemoteURL(w io.Writer, c giteego.PluginComponentScheme) {
	if c.URL == nil {
		return
	}
	if c.URL.GitOps != "" {
		fmt.Fprintf(w, "    remote:  %s\n", c.URL.GitOps)
	}
	if len(c.Params) > 0 {
		parts := make([]string, 0, len(c.Params))
		for k, v := range c.Params {
			parts = append(parts, k+"="+v)
		}
		sort.Strings(parts)
		fmt.Fprintf(w, "    params:  %s\n", strings.Join(parts, "  "))
	}
}

// formatRules returns a compact "required/optional + regex" string.
func formatRules(c giteego.PluginComponentScheme) string {
	var parts []string
	for _, r := range c.Rules {
		if r.Required {
			parts = append(parts, "required")
		} else {
			parts = append(parts, "optional")
		}
		if r.Regex != "" {
			parts = append(parts, "regex="+r.Regex)
		}
	}
	return strings.Join(parts, "  ")
}

// i18nName strips the gitee-go i18n placeholder wrapper (#{{key}}) so output
// shows the key alone instead of the cryptic bracket form; other text passes
// through unchanged.
func i18nName(name string) string {
	name = strings.TrimSpace(name)
	open := strings.Index(name, "#{{")
	close := strings.LastIndex(name, "}}")
	if open >= 0 && close > open {
		return strings.TrimSpace(name[open+3 : close])
	}
	return name
}

// formatDefault renders a component default value: strings pass through (with
// i18n/ref keys stripped), composite values render as compact JSON.
func formatDefault(v interface{}) string {
	switch tv := v.(type) {
	case nil:
		return ""
	case string:
		switch tv {
		case "":
			return ""
		case "true":
			return "true"
		case "false":
			return "false"
		}
		clean := i18nName(tv)
		if strings.HasPrefix(clean, "{{") && strings.HasSuffix(clean, "}}") {
			return clean
		}
		return clean
	default:
		b, err := json.Marshal(tv)
		if err != nil || string(b) == "null" {
			return ""
		}
		return string(b)
	}
}

func newPipelinePluginExampleCmd(f *cmdutil.Factory) *cobra.Command {
	var jobType string

	cmd := &cobra.Command{
		Use:   "example --type <jobType>",
		Short: "Show example pipeline YAML for a plugin",
		Long:  `Show the generated example YAML snippet for a gitee-go plugin job type. The --type value is the plugin's JSON type as shown by 'plugin list'.`,
		Example: `  gitee pipeline plugin example -R owner/repo --type JENKINS_JOB
  gitee pipeline plugin example -R owner/repo --type maven-build@v1.0.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolvePluginClient(f, cmd)
			if err != nil {
				return err
			}
			yaml, err := client.GeneratePluginExampleYaml(f.Context, jobType)
			if err != nil {
				return fmt.Errorf("failed to get plugin example: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "%s\n", yaml)
			return nil
		},
	}

	cmd.Flags().StringVar(&jobType, "type", "", "Plugin JSON job type (from 'plugin list')")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
