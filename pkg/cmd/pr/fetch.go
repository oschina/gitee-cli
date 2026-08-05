package pr

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	gitpkg "gitee.com/oschina/gitee-cli/pkg/git"
)

func newPRFetchCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		branch string
		force  bool
		remote string
	)

	cmd := &cobra.Command{
		Use:   "fetch <number>",
		Short: "Fetch a pull request to a local branch",
		Long: `Fetch a pull request to a local branch without switching to it.

The local branch is named pr_<number> by default. Use --branch to
customize. After fetching, switch with 'git checkout <branch>'.

Unlike 'pr checkout', this command does NOT switch branches.
Use 'pr checkout' for a single-step fetch + switch workflow.`,
		Example: `  gitee pr fetch 42                    # fetch to pr_42
  gitee pr fetch 42 --branch review    # fetch to custom branch
  gitee pr fetch 42 --force            # overwrite existing branch`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			remotes, err := gitpkg.GiteeRemotes()
			if err != nil {
				return err
			}

			selectedRemote, err := pickRemote(remotes, remote)
			if err != nil {
				return err
			}

			localBranch := branch
			if localBranch == "" {
				localBranch = fmt.Sprintf("pr_%d", number)
			}

			if !force {
				if err := checkBranchNotExists(localBranch); err != nil {
					return err
				}
			}

			fmt.Fprintf(f.IOStreams.Out, "Fetching PR #%d from %s into branch %q...\n", number, selectedRemote.Name, localBranch)

			if err := gitpkg.FetchPR(selectedRemote.URL, number, localBranch); err != nil {
				return err
			}

			fmt.Fprintf(f.IOStreams.Out, "✓ Fetched PR #%d → %s\n  Switch with: git checkout %s\n", number, localBranch, localBranch)
			return nil
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Local branch name (default: pr_<number>)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing local branch")
	cmd.Flags().StringVar(&remote, "remote", "", "Git remote name to fetch from (default: first gitee remote)")
	return cmd
}

func pickRemote(remotes []gitpkg.Remote, name string) (gitpkg.Remote, error) {
	if len(remotes) == 0 {
		return gitpkg.Remote{}, fmt.Errorf("no Gitee remotes found; add a remote or run from inside a Gitee repository")
	}
	if name == "" {
		return remotes[0], nil
	}
	for _, r := range remotes {
		if r.Name == name {
			return r, nil
		}
	}
	return gitpkg.Remote{}, fmt.Errorf("remote %q not found among gitee remotes", name)
}

func checkBranchNotExists(branch string) error {
	exists, err := gitpkg.BranchExists(branch)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("branch %q already exists; use --force to overwrite", branch)
	}
	return nil
}
