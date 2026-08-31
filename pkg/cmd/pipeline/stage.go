package pipeline

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineStageCmd returns the `pipeline build stage` command group: the
// runtime stage runs of a build.
func newPipelineStageCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Operate on pipeline stage runs",
		Long: `Inspect and operate on the stage runs (stage builds) of a gitee-go pipeline build.
Stages are runtime entities of a build run: they have no create/config/delete commands
and no standalone existence outside a build. Stage IDs come from build view / build status
output; actions are view/cancel/retry/continue while the owning build is running.`,
	}
	cmd.AddCommand(newPipelineStageViewCmd(f))
	cmd.AddCommand(newPipelineStageCancelCmd(f))
	cmd.AddCommand(newPipelineStageRetryCmd(f))
	cmd.AddCommand(newPipelineStageContinueCmd(f))
	return cmd
}

func stageIDArg(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("stage id is required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid stage id %q", args[0])
	}
	return id, nil
}

// resolveStageClient resolves the repo-scoped gitee-go client for stage ops.
func resolveStageClient(f *cmdutil.Factory, cmd *cobra.Command) (*giteego.Client, error) {
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

// requireStageRunning fetches a stage and errors unless its owning build is
// still running (not a terminal state). Side-effect stage ops are gated on this.
func requireStageRunning(f *cmdutil.Factory, client *giteego.Client, id int64) error {
	st, err := client.GetStageBuild(f.Context, id)
	if err != nil {
		return fmt.Errorf("failed to get stage: %w", err)
	}
	if st == nil {
		return fmt.Errorf("stage %d not found", id)
	}
	build, err := client.GetBuild(f.Context, st.PipelineBuildID)
	if err != nil {
		return fmt.Errorf("failed to get owning build: %w", err)
	}
	if build == nil {
		return fmt.Errorf("owning build %d not found", st.PipelineBuildID)
	}
	if isTerminalBuildStatus(build.Status) {
		return fmt.Errorf("stage %d belongs to build %d which is %s (already finished); stage operations require a running build", id, build.ID, build.Status)
	}
	return nil
}

func newPipelineStageViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "view <stage-id>",
		Short:   "Show a stage",
		Example: `  gitee pipeline build stage view 2278 -R owner/repo`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
			if err != nil {
				return err
			}
			st, err := client.GetStageBuild(f.Context, id)
			if err != nil {
				return fmt.Errorf("failed to get stage: %w", err)
			}
			if st == nil {
				return fmt.Errorf("stage %d not found", id)
			}
			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[giteego.PipelineStageBuildVO](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, st)
				}
				result, err := cmdutil.SelectFields(st, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}
			out := f.IOStreams.Out
			fmt.Fprintf(out, "Stage    %d  [%s]\n", st.ID, st.Status)
			fmt.Fprintf(out, "Name:    %s\n", st.Name)
			fmt.Fprintf(out, "Build:   %d\n", st.PipelineBuildID)
			fmt.Fprintf(out, "Start:   %s\n", timeStr(st.StartTime))
			fmt.Fprintf(out, "End:     %s\n", timeStr(st.EndTime))
			fmt.Fprintf(out, "Duration %s\n", durationStr(st.StartTime, st.EndTime))
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[giteego.PipelineStageBuildVO]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newPipelineStageCancelCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "cancel <stage-id>",
		Short:   "Cancel a stage (requires a running build)",
		Example: `  gitee pipeline build stage cancel 2278 -R owner/repo`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
			if err != nil {
				return err
			}
			if err := requireStageRunning(f, client, id); err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Cancel stage %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.CancelStageBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to cancel stage: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Stage %d cancelled\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineStageRetryCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "retry <stage-id>",
		Short:   "Retry a stage (requires a running build)",
		Example: `  gitee pipeline build stage retry 2278 -R owner/repo`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
			if err != nil {
				return err
			}
			if err := requireStageRunning(f, client, id); err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Retry stage %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.RetryStageBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to retry stage: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Stage %d retried\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineStageContinueCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "continue <stage-id>",
		Short:   "Continue a paused stage",
		Example: `  gitee pipeline build stage continue 2278 -R owner/repo`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
			if err != nil {
				return err
			}
			if err := requireStageRunning(f, client, id); err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Continue stage %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.ContinueStageBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to continue stage: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Stage %d continued\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}
