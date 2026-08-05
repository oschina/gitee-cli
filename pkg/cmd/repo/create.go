package repo

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func newRepoCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name        string
		description string
		homepage    string
		private     bool
		autoInit    bool
		hasIssues   bool
		hasWiki     bool
		jsonOut     bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new repository",
		Long: `Create a new repository under your account.

Supports initializing with a README, enabling Issues and Wiki,
and setting the repository to private.`,
		Example: `  gitee repo create -n my-repo
  gitee repo create -n my-repo -d "Description" --private
  gitee repo create -n my-repo --auto-init --has-issues=false`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return cmdutil.FlagErrorf("--name is required")
			}

			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			r, err := client.CreateRepo(f.Context, &gitee.CreateRepoParams{
				Name:        name,
				Description: description,
				Homepage:    homepage,
				Private:     private,
				HasIssues:   hasIssues,
				HasWiki:     hasWiki,
				AutoInit:    autoInit,
			})
			if err != nil {
				return fmt.Errorf("failed to create repo: %w", err)
			}

			if jsonOut {
				return cmdutil.WriteJSON(f.IOStreams.Out, r)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("repo.created", name, r.HTMLURL))
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Repository name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Repository description")
	cmd.Flags().StringVar(&homepage, "homepage", "", "Repository homepage URL")
	cmd.Flags().BoolVar(&private, "private", false, "Create as private repository")
	cmd.Flags().BoolVar(&autoInit, "auto-init", false, "Initialize with README")
	cmd.Flags().BoolVar(&hasIssues, "has-issues", true, "Enable Issues")
	cmd.Flags().BoolVar(&hasWiki, "has-wiki", true, "Enable Wiki")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}
