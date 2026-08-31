package pipeline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// defaultCommitMessage is used when --message is omitted. It is a git commit
// message stored in the repository, so it stays a fixed English default even
// for localized CLI output (same convention as other CLI defaults).
const defaultCommitMessage = "feat: update pipeline configuration"

// resolveCommitFileName returns the repository-side fileName for a pipeline
// commit. The backend's fileName is a bare file name (e.g.
// "流水线-202608131716.yml") — it does NOT carry a directory prefix — so any
// path given via --file is reduced to its base name. When --file is omitted the
// local yaml file's base name is used (stdin has no name, so --file is then
// required).
func resolveCommitFileName(fileFlag, yamlFile string) string {
	if fileFlag != "" {
		return filepath.Base(fileFlag)
	}
	if yamlFile != "" && yamlFile != "-" {
		return filepath.Base(yamlFile)
	}
	return ""
}

// newPipelineCommitCmd returns the `pipeline commit` command: commit a
// repository pipeline YAML file to a branch via the gitee-code yaml commit
// endpoint. The yaml pipeline commit requires the YAML content — read from a
// local file (--yaml, or "-" for stdin) — which the backend writes under the
// bare file name (--file, default: the local yaml file's base name) on --ref
// with an optional --message.
func newPipelineCommitCmd(f *cmdutil.Factory) *cobra.Command {
	var ref, fileName, yamlFile, message string
	var yes bool

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Commit a repository pipeline YAML file to a branch",
		Long: `Commit a gitee-go repository pipeline YAML file to a branch of the repository.
The YAML content is read from a local file (--yaml, or "-" for stdin) and written
to the repository on --ref with an optional --message. The repository-side
fileName is a bare file name (no directory prefix): it defaults to the base name
of the local yaml file, or is set explicitly with --file. This is a side-effect
operation (creates a git commit in the repository): confirmation is required, or
pass --yes (mandatory in non-interactive mode).`,
		Example: `  gitee pipeline commit -R owner/repo --ref master --yaml ./ci.yml -m "feat: add CI pipeline"
  gitee pipeline commit -R owner/repo --ref develop --file 流水线.yml --yaml ci.yml`,
		Args: cobra.NoArgs,
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

			var data []byte
			if yamlFile == "-" {
				data, err = io.ReadAll(f.IOStreams.In)
			} else {
				data, err = os.ReadFile(yamlFile)
			}
			if err != nil {
				return fmt.Errorf("failed to read yaml file %q: %w", yamlFile, err)
			}
			if strings.TrimSpace(string(data)) == "" {
				return cmdutil.FlagErrorf("yaml file %q is empty", yamlFile)
			}
			fileName = resolveCommitFileName(fileName, yamlFile)
			if fileName == "" {
				return cmdutil.FlagErrorf("fileName is required: pass --file, or use a local --yaml file (stdin has no name)")
			}
			if message == "" {
				message = defaultCommitMessage
			}

			req := giteego.GiteeCodeCommitYamlRequest{
				Branch:        ref,
				FileName:      fileName,
				Content:       string(data),
				CommitMessage: message,
			}

			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf(i18n.T("pipeline.commit.confirm"), fileName, ref, owner+"/"+repo))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}

			if err := client.CommitRepositoryPipelineYaml(f.Context, req); err != nil {
				return fmt.Errorf("failed to commit pipeline yaml: %w", err)
			}

			out := f.IOStreams.Out
			fmt.Fprintln(out, i18n.T("pipeline.commit.done"))
			fmt.Fprintf(out, "Repo:    %s/%s\n", owner, repo)
			fmt.Fprintf(out, "Ref:     %s\n", ref)
			fmt.Fprintf(out, "File:    %s\n", fileName)
			fmt.Fprintf(out, "Message: %s\n", message)
			return nil
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Git branch to commit to (required)")
	_ = cmd.MarkFlagRequired("ref")
	cmd.Flags().StringVar(&fileName, "file", "", "Repository-side pipeline YAML file name (bare name, no directory prefix; default: base name of --yaml)")
	cmd.Flags().StringVar(&yamlFile, "yaml", "", "Local YAML file with the pipeline content ('-' for stdin) (required)")
	_ = cmd.MarkFlagRequired("yaml")
	cmd.Flags().StringVarP(&message, "message", "m", "", fmt.Sprintf("Commit message (default: %q)", defaultCommitMessage))
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}
