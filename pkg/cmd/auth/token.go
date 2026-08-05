package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newTokenCmd(f *cmdutil.Factory) *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the authentication token",
		Long: `Print the stored Personal Access Token for the current or specified host.

Warning: token output is visible in terminal history and logs. Avoid
piping to files or shell history, especially in shared environments.`,
		Example: `  gitee auth token
  gitee auth token --hostname git.company.com`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = f.Hostname
			}
			token, err := config.TokenForHost(hostname)
			if err != nil {
				return err
			}
			if f.IOStreams.IsTerminal() {
				fmt.Fprintln(f.IOStreams.ErrOut, "Warning: token output is plaintext. Avoid piping to logs or shell history.")
			}
			fmt.Fprintln(f.IOStreams.Out, token)
			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "", "Hostname to get token for (default: gitee.com)")
	return cmd
}
