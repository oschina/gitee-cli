package pr

import (
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func resolveRepo(cmd *cobra.Command) (owner, repo string, err error) {
	return cmdutil.ResolveRepo(cmd)
}
