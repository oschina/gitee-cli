package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineExampleCmd returns the `pipeline example` command: print the
// generated example YAML for a whole repository pipeline. It uses the
// repo-scoped client like the other repo pipeline commands (gateway still
// routes through the repo path segment).
func newPipelineExampleCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "example",
		Short: "Show an example repository pipeline YAML",
		Long: `Show the generated example YAML for a whole gitee-go repository pipeline
(version/name/displayName, stages with steps, triggers, notify, strategy and
variables) from the backend. Unlike 'pipeline plugin example' — which prints a
single plugin job snippet — this is a complete pipeline configuration you can
adapt and commit with 'pipeline commit'.`,
		Example: `  gitee pipeline example -R owner/repo`,
		Args:    cobra.NoArgs,
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
			yaml, err := client.GetPipelineYamlExample(f.Context)
			if err != nil {
				return fmt.Errorf("failed to get pipeline yaml example: %w", err)
			}
			if yaml == "" {
				return fmt.Errorf("pipeline yaml example returned no content")
			}
			fmt.Fprintf(f.IOStreams.Out, "%s\n", yaml)
			return nil
		},
	}
	return cmd
}
