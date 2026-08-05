package auth

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

const tokenCreateURL = "https://gitee.com/profile/personal_access_tokens"

func newLoginCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		tokenFromStdin bool
		hostname       string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Gitee",
		Long: `Log in to Gitee with a Personal Access Token.

Create a token at https://gitee.com/profile/personal_access_tokens
then pass it via --with-token or the interactive prompt.

Multi-host: log in to private Gitee instances with --hostname:
  gitee auth login --hostname git.company.com`,
		Example: `  gitee auth login
  printf '%s' "$GITEE_TOKEN" | gitee auth login --with-token
  gitee auth login --hostname git.company.com`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = f.Hostname
			}

			var token string
			if tokenFromStdin {
				const maxTokenBytes = 64 * 1024
				data, err := io.ReadAll(io.LimitReader(f.IOStreams.In, maxTokenBytes+1))
				if err != nil {
					return fmt.Errorf("read token from stdin: %w", err)
				}
				if len(data) > maxTokenBytes {
					return cmdutil.FlagErrorf("token exceeds the maximum supported length")
				}
				token = strings.TrimSpace(string(data))
			}
			if token == "" {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("--with-token is required in non-interactive mode")
				}
				token = interactiveLogin(f, hostname)
			}

			if token == "" {
				return cmdutil.FlagErrorf("token cannot be empty")
			}

			fmt.Fprintln(f.IOStreams.Out, i18n.T("auth.validating"))
			apiPrefix := config.APIPrefixForHost(hostname)
			client := gitee.NewClient(token, gitee.WithBaseURL(apiPrefix))
			user, err := client.GetAuthenticatedUser(f.Context)
			if err != nil {
				return fmt.Errorf("token validation failed: %w", err)
			}

			if hostname == "" || hostname == config.DefaultHost {
				if err := config.SaveToken(token); err != nil {
					return fmt.Errorf("failed to save token: %w", err)
				}
			} else {
				if err := config.SaveHostConfig(hostname, token, apiPrefix); err != nil {
					return fmt.Errorf("failed to save token: %w", err)
				}
			}

			fmt.Fprint(f.IOStreams.Out, i18n.Tf("auth.logged_in",
				func() string {
					if hostname == "" {
						return config.DefaultHost
					}
					return hostname
				}(),
				user.Login, user.Name))
			return nil
		},
	}

	cmd.Flags().BoolVar(&tokenFromStdin, "with-token", false, "Read the personal access token from standard input")
	cmd.Flags().StringVar(&hostname, "hostname", "", "Gitee hostname (default: gitee.com)")
	return cmd
}

func interactiveLogin(f *cmdutil.Factory, hostname string) string {
	createURL := tokenCreateURL
	if hostname != "" && hostname != config.DefaultHost {
		createURL = fmt.Sprintf("https://%s/profile/personal_access_tokens", hostname)
	}

	host := config.DefaultHost
	if hostname != "" && hostname != config.DefaultHost {
		host = hostname
	}

	fmt.Fprintln(f.IOStreams.Out, "")
	fmt.Fprint(f.IOStreams.Out, i18n.Tf("auth.need_token", host)+"\n")

	choice, err := cmdutil.AskSelect(i18n.T("auth.how_to_auth"), []string{
		i18n.T("auth.open_browser"),
		i18n.T("auth.paste_token"),
	}, f.IsTUI())
	if err != nil {
		return ""
	}

	if choice == i18n.T("auth.open_browser") {
		fmt.Fprint(f.IOStreams.Out, i18n.Tf("auth.opening_url", createURL))
		if err := browser.OpenURL(createURL); err != nil {
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("auth.browser_failed", createURL))
		}
		fmt.Fprintln(f.IOStreams.Out, "")
	}

	token, err := cmdutil.AskPassword(i18n.T("auth.paste_token_prompt"), f.IsTUI())
	if err != nil {
		return ""
	}
	return token
}
