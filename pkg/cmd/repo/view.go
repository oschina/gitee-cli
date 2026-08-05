package repo

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func newRepoViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string

	cmd := &cobra.Command{
		Use:   "view [owner/repo]",
		Short: "View a repository",
		Long:  `View the details of a repository, including description, URL, stars, forks, and open issues.`,
		Example: `  gitee repo view owner/repo
  gitee repo view --json=full_name,description,language,stargazers_count`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner, repoName string
			var err error
			if len(args) == 1 {
				parts := splitOwnerRepo(args[0])
				if parts == nil {
					return fmt.Errorf("invalid format, expected owner/repo")
				}
				owner, repoName = parts[0], parts[1]
			} else {
				owner, repoName, err = cmdutil.ResolveRepo(cmd)
				if err != nil {
					return err
				}
			}

			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			r, err := client.GetRepo(f.Context, owner, repoName)
			if err != nil {
				return fmt.Errorf("failed to get repo: %w", err)
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Repository](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, r)
				}
				result, err := cmdutil.SelectFields(r, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			if f.IsTUI() {
				title := fmt.Sprintf("%s — %s", r.FullName, visibilityLabel(r))
				return tui.RunPager(title, repoMarkdown(r), tui.ContentMarkdown)
			}

			fmt.Fprintf(f.IOStreams.Out, "%s\n", r.FullName)
			fmt.Fprintf(f.IOStreams.Out, "Description: %s\n", r.Description)
			fmt.Fprintf(f.IOStreams.Out, "URL:         %s\n", r.HTMLURL)
			fmt.Fprintf(f.IOStreams.Out, "SSH:         %s\n", r.SSHURL)
			fmt.Fprintf(f.IOStreams.Out, "Default:     %s\n", r.DefaultBranch)
			fmt.Fprintf(f.IOStreams.Out, "Stars: %d  Forks: %d  Issues: %d\n", r.StargazersCount, r.ForksCount, r.OpenIssuesCount)
			return nil
		},
	}

	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Repository]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func repoMarkdown(r *gitee.Repository) string {
	var sb strings.Builder

	desc := r.Description
	if desc == "" {
		desc = "_No description provided._"
	}
	sb.WriteString(desc)
	sb.WriteString("\n\n")

	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("- **URL** %s\n", r.HTMLURL))
	sb.WriteString(fmt.Sprintf("- **SSH** `%s`\n", r.SSHURL))
	sb.WriteString(fmt.Sprintf("- **Default branch** `%s`\n", r.DefaultBranch))
	if r.Language != "" {
		sb.WriteString(fmt.Sprintf("- **Language** %s\n", r.Language))
	}
	if r.License != "" {
		sb.WriteString(fmt.Sprintf("- **License** %s\n", r.License))
	}
	sb.WriteString(fmt.Sprintf("- **Visibility** %s\n", visibilityLabel(r)))
	if r.Fork {
		sb.WriteString("- **Fork** yes\n")
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("⭐ %d stars  🍴 %d forks  👁 %d watchers  🐛 %d open issues\n",
		r.StargazersCount, r.ForksCount, r.WatchersCount, r.OpenIssuesCount))

	sb.WriteString("\n---\n\n")

	var features []string
	if r.HasIssues {
		features = append(features, "Issues")
	}
	if r.HasWiki {
		features = append(features, "Wiki")
	}
	if r.HasPage {
		features = append(features, "Pages")
	}
	if r.PullRequestsEnabled {
		features = append(features, "Pull Requests")
	}
	if len(features) > 0 {
		sb.WriteString(fmt.Sprintf("**Features:** %s\n\n", strings.Join(features, " · ")))
	}

	sb.WriteString(fmt.Sprintf("**Owner:** %s  \n", r.Owner.Login))
	sb.WriteString(fmt.Sprintf("**Created:** %s  \n", r.CreatedAt.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Updated:** %s  \n", r.UpdatedAt.Format("2006-01-02")))
	if !r.PushedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("**Last push:** %s\n", r.PushedAt.Format("2006-01-02")))
	}

	return sb.String()
}

func visibilityLabel(r *gitee.Repository) string {
	if r.Private {
		return "private"
	}
	if r.Internal {
		return "internal"
	}
	return "public"
}

func splitOwnerRepo(s string) []string {
	for i, c := range s {
		if c == '/' {
			if i > 0 && i < len(s)-1 {
				return []string{s[:i], s[i+1:]}
			}
			return nil
		}
	}
	return nil
}
