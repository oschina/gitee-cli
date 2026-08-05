package ai

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	pkgai "gitee.com/oschina/gitee-cli/pkg/ai"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func NewAICmd(f *cmdutil.Factory) *cobra.Command {
	var noStream bool
	var chat bool

	cmd := &cobra.Command{
		Use:   "ai [context]",
		Short: "Chat or interact with AI",
		Long: `Ask the AI anything in a single shot (streaming by default) or start a multi-turn chat session with --chat.

Requires ai.base_url, ai.model, and ai.token to be configured.`,
		Example: `  gitee ai "what is a rebase?"
  gitee ai -n "summarise this diff: $(git diff HEAD~1)"
  gitee ai --chat`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			aiCfg, err := config.AI()
			if err != nil {
				return err
			}
			client := pkgai.NewClient(aiCfg.BaseURL, aiCfg.Token, aiCfg.Model)
			ctx := f.Context

			if chat {
				pkgai.Chat(ctx, client, f.IOStreams.In, f.IOStreams.Out)
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("%s", i18n.T("ai.usage_hint"))
			}
			userInput := args[0]

			if noStream {
				reply, err := client.Complete(ctx, "", userInput)
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, reply)
				return nil
			}

			msgs := []pkgai.Message{{Role: "user", Content: userInput}}
			_, err = client.CompleteStream(ctx, msgs, func(chunk string) {
				fmt.Fprint(f.IOStreams.Out, chunk)
			})
			fmt.Fprintln(f.IOStreams.Out)
			return err
		},
	}

	cmd.Flags().BoolVarP(&noStream, "no-stream", "n", false, "Execute without stream")
	cmd.Flags().BoolVarP(&chat, "chat", "c", false, "Execute as chat")
	return cmd
}
