package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/giteego"
)

// newPipelineProgramStageCmd returns the `pipeline program build stage` command
// group for project (enterprise) pipeline stage runs. Like the repo-side stage
// commands, these operate on stage builds by id, but they are located by -E/-P
// instead of -R and hit the /multi-source gateway segment via the program client.
func newPipelineProgramStageCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Operate on program pipeline stage runs",
		Long: `Inspect and operate on the stage runs (stage builds) of a gitee-go program pipeline build.
Stages are runtime entities of a build run: they have no create/config/delete commands
and no standalone existence outside a build. Stage IDs come from build view / build status
output; actions are view/cancel/retry/continue while the owning build is running.`,
	}
	cmd.AddCommand(newPipelineProgramStageViewCmd(f))
	cmd.AddCommand(newPipelineProgramStageCancelCmd(f))
	cmd.AddCommand(newPipelineProgramStageRetryCmd(f))
	cmd.AddCommand(newPipelineProgramStageContinueCmd(f))
	return cmd
}

// requireProgramStageRunning fetches a program stage and errors unless its
// owning build is still running (not a terminal state). Side-effect stage ops
// are gated on this, mirroring requireStageRunning.
func requireProgramStageRunning(f *cmdutil.Factory, client *giteego.Client, id int64) error {
	st, err := client.GetProgramStageBuild(f.Context, id)
	if err != nil {
		return fmt.Errorf("failed to get stage: %w", err)
	}
	if st == nil {
		return fmt.Errorf("stage %d not found", id)
	}
	build, err := client.GetProgramBuild(f.Context, st.PipelineBuildID)
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

func newPipelineProgramStageViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:     "view <stage-id>",
		Short:   "Show a program pipeline stage",
		Example: `  gitee pipeline program build stage view 2278 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			st, err := client.GetProgramStageBuild(f.Context, id)
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

func newPipelineProgramStageCancelCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "cancel <stage-id>",
		Short:   "Cancel a program pipeline stage (requires a running build)",
		Example: `  gitee pipeline program build stage cancel 2278 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if err := requireProgramStageRunning(f, client, id); err != nil {
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
			if err := client.CancelProgramStageBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to cancel stage: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Stage %d cancelled\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineProgramStageRetryCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "retry <stage-id>",
		Short:   "Retry a program pipeline stage (requires a running build)",
		Example: `  gitee pipeline program build stage retry 2278 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if err := requireProgramStageRunning(f, client, id); err != nil {
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
			if err := client.RetryProgramStageBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to retry stage: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Stage %d retried\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineProgramStageContinueCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "continue <stage-id>",
		Short:   "Continue a paused program pipeline stage",
		Example: `  gitee pipeline program build stage continue 2278 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := stageIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if err := requireProgramStageRunning(f, client, id); err != nil {
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
			if err := client.ContinueProgramStageBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to continue stage: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Stage %d continued\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}
