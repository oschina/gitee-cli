package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineProgramPluginCmd returns the `pipeline program plugin` command
// group for project (enterprise) pipelines. These reuse the repo plugin view
// helpers (printSchemeConfig, i18nName, formatDefault) but route through the
// /multi-source segment via the program client.
func newPipelineProgramPluginCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage gitee-go program plugins",
		Long:  `List gitee-go program plugins and inspect their parameter schemes / example YAML.`,
	}
	cmd.AddCommand(newPipelineProgramPluginListCmd(f))
	cmd.AddCommand(newPipelineProgramPluginSchemeCmd(f))
	cmd.AddCommand(newPipelineProgramPluginExampleCmd(f))
	return cmd
}

func newPipelineProgramPluginListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available gitee-go program plugins",
		Long:  `List the gitee-go plugins available to the project, grouped by category.`,
		Example: `  gitee pipeline program plugin list -E 2 -P 423
  gitee pipeline program plugin list -E 2 -P 423 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			plugins, err := client.ListProgramPlugins(f.Context)
			if err != nil {
				return fmt.Errorf("failed to list program plugins: %w", err)
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

func newPipelineProgramPluginSchemeCmd(f *cmdutil.Factory) *cobra.Command {
	var jobType string
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "scheme --type <jobType>",
		Short: "Show a program plugin's parameter scheme",
		Long:  `Show the parameter scheme (fields, defaults, validation) of a gitee-go program plugin for a job type, as shown by 'program plugin list'.`,
		Example: `  gitee pipeline program plugin scheme -E 2 -P 423 --type JENKINS_JOB
  gitee pipeline program plugin scheme -E 2 -P 423 --type maven-build@v1.0.0 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			schemes, err := client.GetProgramPluginSchemes(f.Context, jobType)
			if err != nil {
				return fmt.Errorf("failed to get program plugin scheme: %w", err)
			}
			if len(schemes) == 0 {
				return fmt.Errorf("no scheme for job type %q", jobType)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PluginSchemeVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, schemes)
				}
				result := make(map[string]any, len(schemes))
				for name, scheme := range schemes {
					selected, err := cmdutil.SelectFields(scheme, fields)
					if err != nil {
						return err
					}
					result[name] = selected
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}
			return printProjectSchemes(f, schemes)
		},
	}
	cmd.Flags().StringVar(&jobType, "type", "", "Plugin JSON job type (from 'program plugin list')")
	_ = cmd.MarkFlagRequired("type")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PluginSchemeVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

// printProjectSchemes renders each plugin scheme (keyed by name) returned for a
// job type. The multi-source route typically yields one scheme per job type,
// but the map form supports several.
func printProjectSchemes(f *cmdutil.Factory, schemes map[string]*giteego.PluginSchemeVO) error {
	out := f.IOStreams.Out
	for _, scheme := range schemes {
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
		if len(scheme.Config) > 0 {
			fmt.Fprintln(out)
			printSchemeConfig(out, scheme.Config)
		}
		fmt.Fprintln(out)
	}
	return nil
}

func newPipelineProgramPluginExampleCmd(f *cmdutil.Factory) *cobra.Command {
	var jobType string
	cmd := &cobra.Command{
		Use:   "example --type <jobType>",
		Short: "Show example pipeline YAML for a program plugin",
		Long:  `Show the generated example YAML snippet for a gitee-go program plugin job type.`,
		Example: `  gitee pipeline program plugin example -E 2 -P 423 --type JENKINS_JOB
  gitee pipeline program plugin example -E 2 -P 423 --type maven-build@v1.0.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			yaml, err := client.GenerateProgramPluginExampleYaml(f.Context, jobType)
			if err != nil {
				return fmt.Errorf("failed to get program plugin example: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "%s\n", yaml)
			return nil
		},
	}
	cmd.Flags().StringVar(&jobType, "type", "", "Plugin JSON job type (from 'program plugin list')")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
