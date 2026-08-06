package update

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/build"
	internalupdate "gitee.com/oschina/gitee-cli/internal/update"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

type commandDependencies struct {
	check   func(string) (*internalupdate.ReleaseInfo, error)
	detect  func(context.Context) (internalupdate.Installation, error)
	apply   func(context.Context, *internalupdate.ReleaseInfo, internalupdate.Installation, io.Writer, io.Writer) (internalupdate.InstallResult, error)
	pending func(string) error
}

func NewUpdateCmd(f *cmdutil.Factory) *cobra.Command {
	return newUpdateCmd(f, commandDependencies{
		check:   internalupdate.CheckForUpdate,
		detect:  internalupdate.DetectInstallation,
		apply:   internalupdate.ApplyUpdate,
		pending: internalupdate.ApplyPendingUpdate,
	})
}

func newUpdateCmd(f *cmdutil.Factory, dependencies commandDependencies) *cobra.Command {
	var yes, checkOnly bool
	var applyPath string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update gitee-cli to the latest release",
		Long: `Check for the latest gitee-cli release and update using the detected installation method.

Global npm installations are updated through npm. Release, source-installed,
and standalone binaries are downloaded from Gitee, verified against
checksums.txt, and replaced atomically. The command never invokes sudo.`,
		Example: `  gitee update
  gitee update --check
  gitee update --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if applyPath != "" {
				return dependencies.pending(applyPath)
			}
			if !internalupdate.IsReleaseVersion(build.Version) {
				return cmdutil.FlagErrorf("self-update is unavailable for development build %q", build.Version)
			}

			release, err := dependencies.check(build.Version)
			if err != nil {
				return fmt.Errorf("check for updates: %w", err)
			}
			if release == nil {
				fmt.Fprintf(f.IOStreams.Out, "gitee-cli %s is up to date.\n", build.Version)
				return nil
			}

			installation, err := dependencies.detect(f.Context)
			if err != nil {
				return fmt.Errorf("detect installation method: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "Current version:  %s\n", build.Version)
			fmt.Fprintf(f.IOStreams.Out, "Latest version:   %s\n", release.Version)
			fmt.Fprintf(f.IOStreams.Out, "Installation:     %s\n", installation.Description())
			fmt.Fprintf(f.IOStreams.Out, "Release:          %s\n", release.URL)
			if checkOnly {
				return nil
			}

			if !yes {
				confirmed, err := cmdutil.ConfirmDestructiveAction(f, fmt.Sprintf("Update gitee-cli to %s?", release.Version))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(f.IOStreams.Out, "Update cancelled.")
					return nil
				}
			}

			result, err := dependencies.apply(f.Context, release, installation, f.IOStreams.Out, f.IOStreams.ErrOut)
			if err != nil {
				return err
			}
			if result.Deferred {
				fmt.Fprintf(f.IOStreams.Out, "Update to %s will complete after this process exits.\n", release.Version)
			} else {
				fmt.Fprintf(f.IOStreams.Out, "Updated gitee-cli to %s.\n", release.Version)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for an update without installing it")
	cmd.Flags().StringVar(&applyPath, "apply", "", "internal path used to finish a Windows update")
	_ = cmd.Flags().MarkHidden("apply")
	return cmd
}
