package sshkey

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newSSHKeyDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete an SSH key",
		Long:    `Delete an SSH key by its numeric ID. Use 'ssh-key list' to find the ID.`,
		Aliases: []string{"rm"},
		Example: `  gitee ssh-key delete 42`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid key id: %s", args[0])
			}

			if !yes {
				confirmed, err := cmdutil.ConfirmDestructiveAction(
					f,
					i18n.Tf("sshkey.delete_confirm", id),
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

			if err := client.DeleteSSHKey(f.Context, id); err != nil {
				return fmt.Errorf("failed to delete SSH key %d: %w", id, err)
			}

			fmt.Fprint(f.IOStreams.Out, i18n.Tf("sshkey.deleted", id))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}
