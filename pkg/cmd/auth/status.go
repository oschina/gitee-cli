package auth

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newStatusCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Show authentication status",
		Long:    `Show the current authentication status: which host you're logged in to and your username. Also lists any additional hosts with stored credentials.`,
		Example: `  gitee auth status`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			activeHost := f.Hostname
			if activeHost == "" {
				activeHost = config.DefaultHost
			}

			client, err := f.GiteeClient()
			if err != nil {
				if jsonOut {
					enc := json.NewEncoder(f.IOStreams.Out)
					return enc.Encode(map[string]string{"status": "not logged in", "host": activeHost})
				}
				fmt.Fprint(f.IOStreams.Out, i18n.Tf("auth.not_logged_in", activeHost))
				return nil
			}

			user, err := client.GetAuthenticatedUser(f.Context)
			if err != nil {
				return fmt.Errorf("failed to get user info: %w", err)
			}

			if jsonOut {
				enc := json.NewEncoder(f.IOStreams.Out)
				return enc.Encode(map[string]any{
					"status": "logged in",
					"host":   activeHost,
					"user":   user.Login,
					"name":   user.Name,
				})
			}

			fmt.Fprint(f.IOStreams.Out, i18n.Tf("auth.logged_in", activeHost, user.Login, user.Name))

			otherHosts := config.ListHosts()
			if len(otherHosts) > 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("auth.other_hosts"))
				for _, h := range otherHosts {
					if h != activeHost {
						fmt.Fprintf(f.IOStreams.Out, "  %s\n", h)
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
