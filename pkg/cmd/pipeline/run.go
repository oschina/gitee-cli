package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

func newPipelineRunCmd(f *cmdutil.Factory) *cobra.Command {
	var ref, fileName string
	var params []string
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a repository pipeline build",
		Long:  `Trigger a gitee-go repository pipeline build for a YAML file on a ref (branch/tag).`,
		Example: `  gitee pipeline run -R owner/repo --ref master --file pipeline-example.yml
  gitee pipeline run -R owner/repo --ref release-1.0 --file ci.yml -p KEY=VALUE -p OTHER=x --json`,
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
			// The backend's fileName is a bare file name (no directory prefix).
			// Tolerate an accidental directory prefix (e.g. `.workflow/ci.yml`)
			// by normalizing to the base name, but keep the note visible so the
			// user learns the canonical bare-name form.
			base := filepath.Base(fileName)
			if base != fileName {
				fmt.Fprintf(f.IOStreams.Out, "Note: using file %q (input: %q)\n", base, fileName)
				fileName = base
			}
			req := giteego.BuildRequest{FileName: fileName, Ref: ref}
			for _, kv := range params {
				kv = strings.TrimSpace(kv)
				if kv == "" {
					continue
				}
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("invalid param %q, expected KEY=VALUE", kv)
				}
				req.Params = append(req.Params, giteego.ParamItem{Name: strings.TrimSpace(k), Value: strings.TrimSpace(v)})
			}

			b, err := client.TriggerBuild(f.Context, req)
			if err != nil {
				return fmt.Errorf("failed to trigger build: %w", err)
			}
			if b == nil {
				return fmt.Errorf("trigger build returned no build")
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineBuildVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, b)
				}
				result, err := cmdutil.SelectFields(b, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Build #%d triggered\n", b.BuildNumber)
			fmt.Fprintf(out, "ID:     %d\n", b.ID)
			fmt.Fprintf(out, "Status: %s\n", b.Status)
			fmt.Fprintf(out, "File:   %s\n", fileName)
			fmt.Fprintf(out, "Ref:    %s\n", ref)
			if len(req.Params) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, i18n.T("pipeline.param_in"))
				fmt.Fprintln(out, "----")
				for _, p := range req.Params {
					fmt.Fprintf(out, "  %s=%s\n", p.Name, p.Value)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Git branch or tag (required)")
	_ = cmd.MarkFlagRequired("ref")
	cmd.Flags().StringVar(&fileName, "file", "", "Pipeline YAML file name in the repo root (no directory prefix; required)")
	_ = cmd.MarkFlagRequired("file")
	cmd.Flags().StringArrayVarP(&params, "param", "p", nil, "Build param as KEY=VALUE (repeated)")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}
