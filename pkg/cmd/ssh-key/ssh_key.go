package sshkey

import (
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func NewSSHKeyCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ssh-key",
		Short:   "Manage SSH keys",
		Long:    `List, add, and delete SSH public keys for your Gitee account.`,
		Aliases: []string{"ssh"},
	}

	cmd.AddCommand(newSSHKeyListCmd(f))
	cmd.AddCommand(newSSHKeyAddCmd(f))
	cmd.AddCommand(newSSHKeyDeleteCmd(f))
	return cmd
}
