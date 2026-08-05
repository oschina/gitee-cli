package auth

import (
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func NewAuthCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Gitee",
		Long:  `Log in, log out, and check authentication status for Gitee hosts. Supports multiple hosts via --hostname.`,
	}

	cmd.AddCommand(newLoginCmd(f))
	cmd.AddCommand(newLogoutCmd(f))
	cmd.AddCommand(newStatusCmd(f))
	cmd.AddCommand(newTokenCmd(f))
	return cmd
}
