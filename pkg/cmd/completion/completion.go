package completion

import (
	"github.com/spf13/cobra"
)

func NewCompletionCmd(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion script",
		Long:  `Generate shell completion scripts for bash, zsh, fish, or powershell. Source the output in your shell's config file to enable tab completion for gitee commands.`,
		Example: `  gitee completion bash  > /etc/bash_completion.d/gitee
  gitee completion zsh   > /usr/local/share/zsh/site-functions/_gitee
  gitee completion fish  > ~/.config/fish/completions/gitee.fish
  gitee completion powershell > ~/.config/powershell/gitee.ps1`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			}
			return nil
		},
	}
	return cmd
}
