package repo

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newRepoDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete [owner/repo]",
		Short: "Delete a repository",
		Long:  `Delete a repository. This action is irreversible and requires confirmation. Use --yes to skip the confirmation prompt.`,
		Example: `  gitee repo delete owner/repo
  gitee repo delete owner/repo -y`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner, repo string
			var err error
			if len(args) == 1 {
				owner, repo, err = cmdutil.ParseOwnerRepo(args[0])
			} else {
				owner, repo, err = cmdutil.ResolveRepo(cmd)
			}
			if err != nil {
				return err
			}

			if !yes {
				confirmed, err := cmdutil.ConfirmDestructiveAction(
					f,
					i18n.Tf("repo.delete_confirm", fmt.Sprintf("%s/%s", owner, repo)),
				)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}

			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			if err := client.DeleteRepo(f.Context, owner, repo); err != nil {
				return fmt.Errorf("failed to delete repo: %w", err)
			}

			fmt.Fprint(f.IOStreams.Out, i18n.Tf("repo.deleted", fmt.Sprintf("%s/%s", owner, repo)))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}
