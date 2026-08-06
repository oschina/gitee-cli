package search

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
)

func NewSearchCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search repositories, issues, and users",
		Long:  `Search Gitee repositories, issues, and users using the Gitee v5 search API.`,
	}

	cmd.AddCommand(newReposCmd(f))
	cmd.AddCommand(newIssuesCmd(f))
	cmd.AddCommand(newUsersCmd(f))
	return cmd
}

type commonOptions struct {
	sort       string
	order      string
	page       int
	limit      int
	jsonFields string
}

func (o *commonOptions) addFlags(cmd *cobra.Command, resource, sortFields string) {
	cmd.Flags().StringVar(&o.sort, "sort", "", fmt.Sprintf("Sort by: %s (default: best match)", sortFields))
	cmd.Flags().StringVar(&o.order, "order", "desc", "Sort order: asc, desc")
	cmd.Flags().IntVarP(&o.limit, "limit", "l", 20, fmt.Sprintf("Number of %s per page (max 100)", resource))
	cmd.Flags().IntVarP(&o.page, "page", "p", 1, "Page number")
}

func (o *commonOptions) validate(validSorts []string) error {
	if o.page < 1 {
		return cmdutil.FlagErrorf("--page must be at least 1")
	}
	if o.limit < 1 || o.limit > 100 {
		return cmdutil.FlagErrorf("--limit must be between 1 and 100")
	}
	if o.order != "asc" && o.order != "desc" {
		return cmdutil.FlagErrorf("--order must be one of: asc, desc")
	}
	if o.sort != "" && !slices.Contains(validSorts, o.sort) {
		return cmdutil.FlagErrorf("--sort must be one of: %v", validSorts)
	}
	return nil
}
