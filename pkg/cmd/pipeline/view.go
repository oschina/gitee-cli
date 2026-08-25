package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

func newPipelineViewCmd(f *cmdutil.Factory) *cobra.Command {
	var ref, fileName string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "view",
		Short: "View a repository pipeline YAML",
		Long:  `View the gitee-go pipeline YAML configuration of a file for a ref (branch/tag).`,
		Example: `  gitee pipeline view -R owner/repo --ref master --file .gitee/pipelines/ci.yml
  gitee pipeline view -R owner/repo --ref master --file ci.yml --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := resolveOwnerRepo(f, cmd)
			if err != nil {
				return err
			}
			if err := ensureServiceOpen(f, cmd, owner, repo); err != nil {
				return err
			}
			client, err := repoPipelineClientForService(f, owner, repo, giteego.ServiceIPipe)
			if err != nil {
				return err
			}
			p, err := client.GetRepositoryPipeline(f.Context, ref, fileName)
			if err != nil {
				return fmt.Errorf("failed to get pipeline: %w", err)
			}
			if p == nil {
				return fmt.Errorf("pipeline %q not found for ref %q", fileName, ref)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineYamlSummaryVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, p)
				}
				result, err := cmdutil.SelectFields(p, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "File:     %s\n", p.FileName)
			fmt.Fprintf(out, "Ref:      %s\n", ref)
			if p.Name != "" {
				fmt.Fprintf(out, "Name:     %s\n", p.Name)
			}
			if p.Message != "" {
				fmt.Fprintln(out)
				fmt.Fprintf(out, "parse error: %s\n", p.Message)
			}
			if p.Yaml != "" {
				fmt.Fprintln(out)
				fmt.Fprintln(out, i18n.T("pipeline.yaml_config"))
				fmt.Fprintln(out, "--------")
				fmt.Fprintf(out, "%s\n", p.Yaml)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Git branch or tag (required)")
	_ = cmd.MarkFlagRequired("ref")
	cmd.Flags().StringVar(&fileName, "file", "", "Pipeline YAML file name (required)")
	_ = cmd.MarkFlagRequired("file")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineYamlSummaryVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
