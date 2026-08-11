package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newLogoutCmd(f *cmdutil.Factory) *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of Gitee",
		Long: `Remove stored credentials for the current or specified host.

By default logs out of the configured host. Use --hostname to log out of a
specific private instance:
  gitee auth logout --hostname git.company.com`,
		Example: `  gitee auth logout
  gitee auth logout --hostname git.company.com`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = f.Hostname
			}

			if hostname != "" && hostname != config.DefaultHost {
				if err := config.DeleteHostConfig(hostname); err != nil {
					return fmt.Errorf("failed to remove credentials for %s: %w", hostname, err)
				}
				if config.Get(config.KeyHost) == hostname {
					if err := config.Set(config.KeyHost, config.DefaultHost); err != nil {
						return fmt.Errorf("failed to reset default host: %w", err)
					}
				}
				fmt.Fprint(f.IOStreams.Out, i18n.Tf("auth.logged_out", hostname))
				return nil
			}

			if err := config.DeleteToken(); err != nil {
				return fmt.Errorf("failed to remove credentials: %w", err)
			}
			fmt.Fprintln(f.IOStreams.Out, i18n.T("auth.logged_out_default"))
			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "", "Hostname to log out of (default: configured host)")
	return cmd
}
