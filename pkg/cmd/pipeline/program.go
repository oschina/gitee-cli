package pipeline

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// programPipelineIDPrefix is the identifier form used by the pipelineOps
// *build* endpoints ("pipeline.ops.pipeline.{id}"). The pipeline list endpoint
// returns the bare numeric id ("710") instead; pipelineIDFromIdentifier accepts
// both forms.
const programPipelineIDPrefix = "pipeline.ops.pipeline."

// newPipelineProgramCmd returns the `pipeline program` command family for
// program (enterprise) pipelines. Unlike the repo pipeline commands these are
// located by --enterprise/-E and --program/-P (required) instead of -R, and are
// routed through the /multi-source gateway segment:
//
//	{host}/{enterprise}/{program}/gitee-go/ipipe/rest/v5/multi-source/...
func newPipelineProgramCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "program",
		Short: "Manage gitee-go program (enterprise) pipelines",
		Long: `Manage gitee-go program pipelines (pipelineOps / multi-source), which are
scoped to an enterprise and a program rather than a repository.

Program pipelines are addressed by --enterprise/-E <enterprise-id> and
--program/-P <program-id>. A program pipeline's list id is the bare
pipeline id (e.g. 706), while its build history endpoint uses the
"pipeline.ops.pipeline.706" identifier.`,
		Example: `  gitee pipeline program list -E 2 -P 423
  gitee pipeline program run --pipeline 706 -E 2 -P 423
  gitee pipeline program build list --pipeline 706 -E 2 -P 423`,
	}

	cmd.PersistentFlags().StringP("enterprise", "E", "", "Enterprise id (required)")
	cmd.PersistentFlags().StringP("program", "P", "", "Program id (required)")

	cmd.AddCommand(newPipelineProgramListCmd(f))
	cmd.AddCommand(newPipelineProgramViewCmd(f))
	cmd.AddCommand(newPipelineProgramCreateCmd(f))
	cmd.AddCommand(newPipelineProgramEditCmd(f))
	cmd.AddCommand(newPipelineProgramCloneCmd(f))
	cmd.AddCommand(newPipelineProgramDeleteCmd(f))
	cmd.AddCommand(newPipelineProgramHistoryCmd(f))
	cmd.AddCommand(newPipelineProgramHistoryApplyCmd(f))
	cmd.AddCommand(newPipelineProgramRunCmd(f))
	cmd.AddCommand(newPipelineProgramBuildCmd(f))
	cmd.AddCommand(newPipelineProgramParamCmd(f))
	cmd.AddCommand(newPipelineProgramTemplateCmd(f))
	cmd.AddCommand(newPipelineProgramPluginCmd(f))
	return cmd
}

// resolveProgram returns the enterprise/program pair required by program
// pipeline commands. The identifier is reused by pipelineIDArg for ids taken
// from the -E/-P location.
func resolveProgram(_ *cmdutil.Factory, cmd *cobra.Command) (string, string, error) {
	ent, _ := cmd.Flags().GetString("enterprise")
	prog, _ := cmd.Flags().GetString("program")
	if ent == "" {
		return "", "", fmt.Errorf("enterprise id is required: use --enterprise/-E <id>")
	}
	if prog == "" {
		return "", "", fmt.Errorf("program id is required: use --program/-P <id>")
	}
	return ent, prog, nil
}

// programPipelineClientForService builds a gitee-go client for a service
// scoped to an enterprise/program pair (pathBase "{enterprise}/{program}").
func programPipelineClientForService(f *cmdutil.Factory, ent, prog, service string) (*giteego.Client, error) {
	c, err := f.GoAPIClient(ent+"/"+prog, service)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// programClient resolves the -E/-P location and returns the ipipe client.
func programClient(f *cmdutil.Factory, cmd *cobra.Command) (*giteego.Client, string, string, error) {
	ent, prog, err := resolveProgram(f, cmd)
	if err != nil {
		return nil, "", "", err
	}
	c, err := programPipelineClientForService(f, ent, prog, giteego.ServiceIPipe)
	if err != nil {
		return nil, "", "", err
	}
	return c, ent, prog, nil
}

// pipelineIDArg resolves a numeric pipeline id argument, accepting either a
// bare number or a "pipeline.ops.pipeline.{id}" build identifier.
func pipelineIDArg(arg string) (int64, error) {
	if arg == "" {
		return 0, fmt.Errorf("pipeline id is required")
	}
	if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
		return id, nil
	}
	if id, ok := pipelineIDFromIdentifier(arg); ok {
		return id, nil
	}
	return 0, fmt.Errorf("invalid pipeline id %q (expected a numeric id or a pipeline.ops.pipeline.* identifier)", arg)
}

// pipelineIDFromIdentifier parses the numeric id of a program pipeline
// identifier. The pipelineOps list endpoint returns the bare id ("710"),
// while the build endpoints use the "pipeline.ops.pipeline.{id}" form; both
// are accepted. Returns (0, false) when the identifier is not a pipeline id.
func pipelineIDFromIdentifier(identifier string) (int64, bool) {
	if strings.HasPrefix(identifier, programPipelineIDPrefix) {
		identifier = strings.TrimPrefix(identifier, programPipelineIDPrefix)
	}
	id, err := strconv.ParseInt(identifier, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// pipelineIDOf extracts the numeric id from a program pipeline VO by parsing
// its identifier. Returns 0 when it cannot be derived.
func pipelineIDOf(p *giteego.PipelineVO) int64 {
	if p == nil {
		return 0
	}
	id, _ := pipelineIDFromIdentifier(p.Identifier)
	return id
}

// pipelineFlagID returns the value of the required --pipeline flag.
func pipelineFlagID(cmd *cobra.Command) (int64, error) {
	v, _ := cmd.Flags().GetInt64("pipeline")
	if v <= 0 {
		return 0, fmt.Errorf("--pipeline <pipeline-id> is required (run 'gitee pipeline program list' to see ids)")
	}
	return v, nil
}
