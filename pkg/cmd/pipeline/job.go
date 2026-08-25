package pipeline

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

// newPipelineJobCmd returns the `pipeline build job` command group: the
// runtime job runs of a build.
func newPipelineJobCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Operate on pipeline job runs",
		Long: `Operate on the job runs (job builds) of a gitee-go pipeline build.
Jobs are runtime entities of a build run: they have no create/config/delete commands
and no standalone existence outside a build. Job IDs come from build view / build status
output; actions are cancel/skip/retry/mark-success while the owning build is running.`,
	}
	cmd.AddCommand(newPipelineJobCancelCmd(f))
	cmd.AddCommand(newPipelineJobSkipCmd(f))
	cmd.AddCommand(newPipelineJobRetryCmd(f))
	cmd.AddCommand(newPipelineJobMarkSuccessCmd(f))
	return cmd
}

func jobIDArg(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("job id is required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid job id %q", args[0])
	}
	return id, nil
}

func newPipelineJobCancelCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "cancel <job-id>",
		Short:   "Cancel a job (requires a running build)",
		Example: `  gitee pipeline build job cancel 2730`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
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
			if err := client.CancelJobBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to cancel job: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Job %d cancelled\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineJobSkipCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "skip <job-id>",
		Short:   "Skip a job (requires a running build)",
		Example: `  gitee pipeline build job skip 2730`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
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
			if err := client.SkipJobBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to skip job: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Job %d skipped\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineJobRetryCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "retry <job-id>",
		Short:   "Retry a job (requires a running build)",
		Example: `  gitee pipeline build job retry 2730`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
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
			if err := client.RetryJobBuild(f.Context, id); err != nil {
				return fmt.Errorf("failed to retry job: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Job %d retried\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newPipelineJobMarkSuccessCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	var reason string
	cmd := &cobra.Command{
		Use:     "mark-success <job-id>",
		Short:   "Mark a job as success (requires a running build)",
		Example: `  gitee pipeline build job mark-success 2730`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := jobIDArg(args)
			if err != nil {
				return err
			}
			client, err := resolveStageClient(f, cmd)
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
			if err := client.MarkJobAsSuccess(f.Context, id, reason); err != nil {
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
