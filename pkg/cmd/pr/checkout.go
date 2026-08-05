package pr

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	gitpkg "gitee.com/oschina/gitee-cli/pkg/git"
)

func newPRCheckoutCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		branch string
		force  bool
		remote string
	)

	cmd := &cobra.Command{
		Use:   "checkout <number>",
		Short: "Check out a pull request locally",
		Long: `Fetch a pull request to a local branch and switch to it.

The local branch is named pr_<number> by default. Use --branch to
customize.

Unlike 'pr fetch', this command also switches to the branch after
fetching. If the branch already exists, it just switches to it
without fetching again (use --force to re-fetch).`,
		Example: `  gitee pr checkout 42
  gitee pr checkout 42 --branch review
  gitee pr checkout 42 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			localBranch := branch
			if localBranch == "" {
				localBranch = fmt.Sprintf("pr_%d", number)
			}

			exists, err := gitpkg.BranchExists(localBranch)
			if err != nil {
				return err
			}

			if exists && !force {
				fmt.Fprintf(f.IOStreams.Out, "Branch %q already exists, switching to it\n", localBranch)
				fmt.Fprintf(f.IOStreams.Out, "  Use --force to re-fetch from remote\n")
				return gitpkg.Checkout(localBranch)
			}

			remotes, err := gitpkg.GiteeRemotes()
			if err != nil {
				return err
			}
			selectedRemote, err := pickRemote(remotes, remote)
			if err != nil {
				return err
			}

			fmt.Fprintf(f.IOStreams.Out, "Fetching PR #%d from %s into branch %q...\n", number, selectedRemote.Name, localBranch)
			if err := gitpkg.FetchPR(selectedRemote.URL, number, localBranch); err != nil {
				return err
			}

			fmt.Fprintf(f.IOStreams.Out, "Switching to branch %q\n", localBranch)
			if err := gitpkg.Checkout(localBranch); err != nil {
				return err
			}

			fmt.Fprintf(f.IOStreams.Out, "✓ Now on PR #%d — %s\n", number, localBranch)
			return nil
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Local branch name (default: pr_<number>)")
	cmd.Flags().BoolVar(&force, "force", false, "Re-fetch even if branch already exists")
	cmd.Flags().StringVar(&remote, "remote", "", "Git remote to fetch from (default: first gitee remote)")
	return cmd
}
