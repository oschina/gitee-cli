package repo

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func newRepoCloneCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		useSSH   bool
		branch   string
		depthRaw string
		dir      string
	)

	cmd := &cobra.Command{
		Use:   "clone <owner/repo>",
		Short: "Clone a repository",
		Long: `Clone a repository from Gitee.

By default, clones via HTTPS with token authentication (auto-detected).
Use --ssh to clone via SSH instead.

Supports full URLs (https://... or git@...) as well as owner/repo format.
When using owner/repo format, the hostname is inferred from --hostname
or the default gitee.com.`,
		Example: `  gitee repo clone owner/repo
  gitee repo clone owner/repo --ssh
  gitee repo clone owner/repo --branch develop --depth 1
  gitee repo clone https://github.com/owner/repo.git`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]

			var cloneURL string
			var cloneEnv []string
			if strings.Contains(input, "://") || strings.HasPrefix(input, "git@") {
				cloneURL = input
			} else {
				parts := splitOwnerRepo(input)
				if parts == nil {
					return fmt.Errorf("invalid format, expected owner/repo")
				}
				hostname := f.Hostname
				if hostname == "" {
					hostname = config.DefaultHost
				}
				if useSSH {
					cloneURL = fmt.Sprintf("git@%s:%s/%s.git", hostname, parts[0], parts[1])
				} else {
					token, err := config.TokenForHost(hostname)
					if err == nil && token != "" {
						cloneEnv = gitHTTPSAuthEnv(hostname, token)
					}
					cloneURL = fmt.Sprintf("https://%s/%s/%s.git", hostname, parts[0], parts[1])
				}
			}

			gitArgs := []string{"clone", cloneURL}
			if branch != "" {
				gitArgs = append(gitArgs, "--branch", branch)
			}
			if depthRaw != "" {
				depth, err := strconv.Atoi(depthRaw)
				if err != nil || depth <= 0 {
					return cmdutil.FlagErrorf("--depth must be a positive integer, got %q", depthRaw)
				}
				gitArgs = append(gitArgs, "--depth", strconv.Itoa(depth))
			}
			if dir != "" {
				gitArgs = append(gitArgs, dir)
			}

			c := exec.Command("git", gitArgs...)
			if cloneEnv != nil {
				c.Env = cloneEnv
			}
			c.Stdin = f.IOStreams.In
			c.Stdout = f.IOStreams.Out
			c.Stderr = f.IOStreams.ErrOut
			return c.Run()
		},
	}

	cmd.Flags().BoolVar(&useSSH, "ssh", false, "Clone using SSH instead of HTTPS")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Clone a specific branch")
	cmd.Flags().StringVar(&depthRaw, "depth", "", "Create a shallow clone with the specified commit depth")
	cmd.Flags().StringVar(&dir, "dir", "", "Clone into this directory instead of the default")
	return cmd
}

// gitHTTPSAuthEnv passes an HTTP Basic header through Git's environment-only
// config. The token is kept out of process arguments and the cloned remote URL.
func gitHTTPSAuthEnv(hostname, token string) []string {
	count := 0
	if raw := os.Getenv("GIT_CONFIG_COUNT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			count = parsed
		}
	}
	header := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("oauth2:"+token))
	return append(os.Environ(),
		fmt.Sprintf("GIT_CONFIG_KEY_%d=http.https://%s/.extraHeader", count, hostname),
		fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", count, header),
		fmt.Sprintf("GIT_CONFIG_COUNT=%d", count+1),
		"GIT_TERMINAL_PROMPT=0",
	)
}
