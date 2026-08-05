package repo

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newRepoForkCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		org     string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "fork <owner/repo>",
		Short: "Fork a repository",
		Long:  `Fork a repository to your account or to an organization with --org.`,
		Example: `  gitee repo fork owner/repo
  gitee repo fork owner/repo --org my-org`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := splitOwnerRepo(args[0])
			if parts == nil {
				return fmt.Errorf("invalid format, expected owner/repo")
			}

			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			r, err := client.ForkRepo(f.Context, parts[0], parts[1], org)
			if err != nil {
				return fmt.Errorf("failed to fork repo: %w", err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, r)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("repo.forked", r.FullName, r.HTMLURL))
			return nil
		},
	}

	cmd.Flags().StringVar(&org, "org", "", "Fork to organization")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
