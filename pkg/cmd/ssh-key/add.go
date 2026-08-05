package sshkey

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newSSHKeyAddCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title   string
		keyFile string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an SSH key",
		Long: `Add an SSH public key to your Gitee account.

Both --title and --file are required. The file should contain a
public key (e.g., ~/.ssh/id_rsa.pub). Use --file - to read it from stdin.`,
		Example: `  gitee ssh-key add -t "My laptop" -f ~/.ssh/id_rsa.pub
  cat ~/.ssh/id_rsa.pub | gitee ssh-key add -t "CI key" -f -`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return cmdutil.FlagErrorf("--title is required")
			}
			if keyFile == "" {
				return cmdutil.FlagErrorf("--file is required")
			}

			var data []byte
			var err error
			if keyFile == "-" {
				const maxKeyBytes = 1024 * 1024
				data, err = io.ReadAll(io.LimitReader(f.IOStreams.In, maxKeyBytes+1))
				if err == nil && len(data) > maxKeyBytes {
					return cmdutil.FlagErrorf("SSH key exceeds the maximum supported length")
				}
			} else {
				data, err = os.ReadFile(keyFile)
			}
			if err != nil {
				return fmt.Errorf("failed to read SSH key %q: %w", keyFile, err)
			}

			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			key, err := client.AddSSHKey(f.Context, title, string(data))
			if err != nil {
				return fmt.Errorf("failed to add SSH key: %w", err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, key)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("sshkey.added", key.ID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Key title (required)")
	cmd.Flags().StringVarP(&keyFile, "file", "f", "", "Path to public key file, or - for stdin (required)")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
