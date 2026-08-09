package root

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/build"
	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/update"
	cmdai "gitee.com/oschina/gitee-cli/pkg/cmd/ai"
	"gitee.com/oschina/gitee-cli/pkg/cmd/alias"
	"gitee.com/oschina/gitee-cli/pkg/cmd/api"
	"gitee.com/oschina/gitee-cli/pkg/cmd/auth"
	"gitee.com/oschina/gitee-cli/pkg/cmd/completion"
	cmdconfig "gitee.com/oschina/gitee-cli/pkg/cmd/config"
	"gitee.com/oschina/gitee-cli/pkg/cmd/issue"
	"gitee.com/oschina/gitee-cli/pkg/cmd/pr"
	"gitee.com/oschina/gitee-cli/pkg/cmd/release"
	"gitee.com/oschina/gitee-cli/pkg/cmd/repo"
	"gitee.com/oschina/gitee-cli/pkg/cmd/search"
	sshkey "gitee.com/oschina/gitee-cli/pkg/cmd/ssh-key"
	updatecmd "gitee.com/oschina/gitee-cli/pkg/cmd/update"
	"gitee.com/oschina/gitee-cli/pkg/cmd/version"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func NewRootCmd(ctx context.Context, f *cmdutil.Factory) *cobra.Command {
	var noTUI bool
	var hostname string
	var quiet bool
	var verbose bool

	cmd := &cobra.Command{
		Use:           "gitee",
		Short:         "Gitee CLI — work with Gitee from the terminal",
		Long:          `Gitee CLI is a command-line tool for Gitee.`,
		Version:       fmt.Sprintf("%s (%s)", build.Version, build.CommitSHA),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			f.Context = cmd.Context()
			if noTUI {
				f.NoTUI = true
			}
			if hostname != "" {
				f.Hostname = hostname
			}
			if quiet {
				f.IOStreams.SetQuiet()
			}
			if verbose {
				handler := slog.NewTextHandler(f.IOStreams.ErrOut, &slog.HandlerOptions{Level: slog.LevelDebug})
				slog.SetDefault(slog.New(handler))
			}
			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&noTUI, "no-tui", false, "Disable TUI mode")
	cmd.PersistentFlags().StringVar(&hostname, "hostname", "", "Gitee hostname (default: gitee.com)")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all output except errors")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Enable verbose debug output")

	cmd.SetVersionTemplate("gitee version {{.Version}}\n")
	cmd.SetContext(ctx)
	cmd.SetOut(f.IOStreams.Out)
	cmd.SetErr(f.IOStreams.ErrOut)

	cmd.AddCommand(auth.NewAuthCmd(f))
	cmd.AddCommand(cmdconfig.NewConfigCmd(f))
	cmd.AddCommand(issue.NewIssueCmd(f))
	cmd.AddCommand(pr.NewPRCmd(f))
	cmd.AddCommand(repo.NewRepoCmd(f))
	cmd.AddCommand(release.NewReleaseCmd(f))
	cmd.AddCommand(sshkey.NewSSHKeyCmd(f))
	cmd.AddCommand(search.NewSearchCmd(f))
	cmd.AddCommand(updatecmd.NewUpdateCmd(f))
	cmd.AddCommand(version.NewVersionCmd(f))
	cmd.AddCommand(alias.NewAliasCmd(f))
	cmd.AddCommand(api.NewAPICmd(f))
	cmd.AddCommand(cmdai.NewAICmd(f))
	cmd.AddCommand(completion.NewCompletionCmd(cmd))

	return cmd
}

func Execute(ctx context.Context, f *cmdutil.Factory) {
	rootCmd := NewRootCmd(ctx, f)
	args := os.Args[1:]

	if expandedArgs, ok := expandAlias(rootCmd, args); ok {
		args = expandedArgs
		rootCmd.SetArgs(expandedArgs)
	} else if shellCmd, ok := expandShellAlias(rootCmd, args); ok {
		os.Exit(runShellAlias(shellCmd, f.IOStreams))
	}

	var updateCh chan *update.ReleaseInfo
	if !skipUpdateForArgs(args) && update.ShouldCheck(config.UpdateCheckEnabled(), hasQuietFlag(args)) {
		updateCh = make(chan *update.ReleaseInfo, 1)
		go func() {
			info, _ := update.CheckForUpdateCached(build.Version, config.UpdateCachePath(), 24*time.Hour)
			updateCh <- info
		}()
	}

	if err := rootCmd.Execute(); err != nil {
		slog.Debug("command error", "error", err)
		fmt.Fprintln(f.IOStreams.ErrOut, cmdutil.FriendlyError(err))
		os.Exit(1)
	}

	if updateCh != nil {
		select {
		case info := <-updateCh:
			if info != nil {
				fmt.Fprintf(f.IOStreams.ErrOut, "\nA new release of gitee-cli is available: %s → %s\n%s\n",
					build.Version, info.Version, info.URL)
				fmt.Fprintln(f.IOStreams.ErrOut, "Run `gitee update` to install it.")
			}
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func hasQuietFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--quiet" || arg == "-q" {
			return true
		}
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.Contains(arg[1:], "q") {
			return true
		}
	}
	return false
}

func skipUpdateForArgs(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" || arg == "--version" {
			return true
		}
	}
	return args[0] == "help" || args[0] == "version" || args[0] == "completion" || args[0] == "update"
}

func expandAlias(rootCmd *cobra.Command, args []string) ([]string, bool) {
	if len(args) == 0 {
		return nil, false
	}
	name := args[0]
	if _, _, err := rootCmd.Find(args); err == nil {
		return nil, false
	}
	expansion := config.Get("alias." + name)
	if expansion == "" || strings.HasPrefix(expansion, "!") {
		return nil, false
	}

	expansion = strings.TrimPrefix(expansion, rootCmd.Name()+" ")

	positionalVars := regexp.MustCompile(`\$(\d+)`)
	hasVars := positionalVars.MatchString(expansion)

	if hasVars {
		expanded := positionalVars.ReplaceAllStringFunc(expansion, func(match string) string {
			n, _ := strconv.Atoi(match[1:])
			idx := n - 1
			if idx >= 0 && idx < len(args)-1 {
				return args[idx+1]
			}
			return match
		})
		return strings.Fields(expanded), true
	}

	expanded := strings.Fields(expansion)
	return append(expanded, args[1:]...), true
}

// expandShellAlias detects aliases prefixed with "!" and returns the shell
// command string with positional arguments substituted. The caller is
// responsible for executing the command.
func expandShellAlias(rootCmd *cobra.Command, args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	name := args[0]
	if _, _, err := rootCmd.Find(args); err == nil {
		return "", false
	}
	expansion := config.Get("alias." + name)
	if !strings.HasPrefix(expansion, "!") {
		return "", false
	}

	script := expansion[1:]

	positionalVars := regexp.MustCompile(`\$(\d+)`)
	if positionalVars.MatchString(script) {
		script = positionalVars.ReplaceAllStringFunc(script, func(match string) string {
			n, _ := strconv.Atoi(match[1:])
			idx := n - 1
			if idx >= 0 && idx < len(args)-1 {
				return args[idx+1]
			}
			return match
		})
	} else if len(args) > 1 {
		script = script + " " + strings.Join(args[1:], " ")
	}

	return script, true
}

func runShellAlias(script string, ios *iostreams.IOStreams) int {
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell, flag = "cmd", "/c"
	} else {
		shell, flag = "sh", "-c"
	}
	cmd := exec.Command(shell, flag, script)
	cmd.Stdin = ios.In
	cmd.Stdout = ios.Out
	cmd.Stderr = ios.ErrOut
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(ios.ErrOut, "shell alias error: %v\n", err)
		return 1
	}
	return 0
}
