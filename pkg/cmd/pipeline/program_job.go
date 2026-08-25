package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

// newPipelineProgramJobCmd returns the `pipeline program build job` command
// group for project (enterprise) pipeline job runs. Located by -E/-P and routed
// through the /multi-source gateway segment like the other program pipeline
// commands.
func newPipelineProgramJobCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Operate on program pipeline job runs",
		Long: `Operate on the job runs (job builds) of a gitee-go program pipeline build.
Jobs are runtime entities of a build run: they have no create/config/delete commands
and no standalone existence outside a build. Job IDs come from build view / build status
output; actions are cancel/skip/retry/mark-success while the owning build is running.`,
	}
	cmd.AddCommand(newPipelineProgramJobCancelCmd(f))
	cmd.AddCommand(newPipelineProgramJobSkipCmd(f))
	cmd.AddCommand(newPipelineProgramJobRetryCmd(f))
	cmd.AddCommand(newPipelineProgramJobMarkSuccessCmd(f))
	return cmd
}

func newPipelineProgramJobCancelCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "cancel <job-id>",
		Short:   "Cancel a program pipeline job (requires a running build)",
		Example: `  gitee pipeline program build job cancel 2730 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Cancel job %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.CancelProgramJobBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to cancel job: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Job %d cancelled\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineProgramJobSkipCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "skip <job-id>",
		Short:   "Skip a program pipeline job (requires a running build)",
		Example: `  gitee pipeline program build job skip 2730 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Skip job %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.SkipProgramJobBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to skip job: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Job %d skipped\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineProgramJobRetryCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "retry <job-id>",
		Short:   "Retry a program pipeline job (requires a running build)",
		Example: `  gitee pipeline program build job retry 2730 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Retry job %d?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.RetryProgramJobBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to retry job: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Job %d retried\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineProgramJobMarkSuccessCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	var reason string
	cmd := &cobra.Command{
		Use:     "mark-success <job-id>",
		Short:   "Mark a program pipeline job as success (requires a running build)",
		Example: `  gitee pipeline program build job mark-success 2730 -E 2 -P 423`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, _, _, err := programClient(f, cmd)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Mark job %d as success?", id))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}
			if err := client.MarkProgramJobAsSuccess(f.Context, id, reason); err != nil {
				return fmt.Errorf("failed to mark job as success: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Job %d marked as success\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	cmd.Flags().StringVar(&reason, "reason", "", "Optional reason")
	return cmd
}
