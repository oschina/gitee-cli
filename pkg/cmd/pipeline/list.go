package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

func newPipelineListCmd(f *cmdutil.Factory) *cobra.Command {
	var ref string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repository pipeline YAML files",
		Long:  `List the gitee-go pipeline YAML files of the current repository for a ref (branch/tag).`,
		Example: `  gitee pipeline list -R owner/repo --ref master
  gitee pipeline list -R owner/repo --ref develop --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := resolveOwnerRepo(f, cmd)
			if err != nil {
				return err
			}
			client, err := repoPipelineClientForService(f, owner, repo, giteego.ServiceIPipe)
			if err != nil {
				return err
			}
			if err := ensureServiceOpen(f, cmd, owner, repo); err != nil {
				return err
			}
			pipes, err := client.ListRepositoryPipelines(f.Context, ref)
			if err != nil {
				return fmt.Errorf("failed to list pipelines: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineYamlSummaryVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, pipes)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, pipes, fields)
			}

			if len(pipes) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.Tf("pipeline.no_pipeline_files", ref))
				return nil
			}
			rows := [][]string{{"FILE", "NAME", "NOTE"}}
			for _, p := range pipes {
				note := ""
				if p.Message != "" {
					note = "parse error: " + p.Message
				}
				rows = append(rows, []string{p.FileName, p.Name, note})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Git branch or tag (required)")
	_ = cmd.MarkFlagRequired("ref")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineYamlSummaryVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
