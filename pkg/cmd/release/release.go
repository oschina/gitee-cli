package release

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

func NewReleaseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage releases",
		Long:  `List, view, create, edit, and delete releases for the repository.`,
	}
	cmd.PersistentFlags().StringP("repo", "R", "", "owner/repo (default: inferred from git remote)")
	cmd.AddCommand(newReleaseListCmd(f))
	cmd.AddCommand(newReleaseViewCmd(f))
	cmd.AddCommand(newReleaseCreateCmd(f))
	cmd.AddCommand(newReleaseEditCmd(f))
	cmd.AddCommand(newReleaseDeleteCmd(f))
	return cmd
}

func newReleaseListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit      int
		page       int
		jsonFields string
	)
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List releases",
		Long:    `List releases for the repository, including tag name, release name, and creation date.`,
		Aliases: []string{"ls"},
		Example: `  gitee release list
  gitee release list --json=id,tag_name,name,prerelease`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			releases, err := client.ListReleases(f.Context, owner, repo, page, limit)
			if err != nil {
				return fmt.Errorf("failed to list releases: %w", err)
			}
			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Release](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, releases)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, releases, fields)
			}
			if len(releases) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("release.no_results"))
				return nil
			}
			if f.IsTUI() {
				return releaseListTUI(f.Context, releases, owner, repo, f.Hostname, client)
			}
			rows := [][]string{{"ID", "TAG", "NAME", "PRE", "CREATED"}}
			for _, r := range releases {
				pre := ""
				if r.Prerelease {
					pre = "pre"
				}
				rows = append(rows, []string{
					strconv.Itoa(r.ID),
					r.TagName,
					tui.Truncate(r.Name, 30),
					pre,
					r.CreatedAt.Format(time.DateOnly),
				})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of releases per page")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Release]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newReleaseViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "view <id-or-tag>",
		Short: "View a release by numeric ID or tag name",
		Long:  `View the details of a release, including its tag name, release notes, author, and creation date.`,
		Example: `  gitee release view v1.0.0
  gitee release view 42 --json=id,tag_name,name,body,prerelease`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			var r *gitee.Release
			if id, err := strconv.Atoi(args[0]); err == nil {
				r, err = client.GetRelease(f.Context, owner, repo, id)
				if err != nil {
					return fmt.Errorf("failed to get release: %w", err)
				}
			} else {
				r, err = client.GetReleaseByTag(f.Context, owner, repo, args[0])
				if err != nil {
					return fmt.Errorf("failed to get release: %w", err)
				}
			}

			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Release](f.IOStreams.Out)
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
			pre := ""
			if r.Prerelease {
				pre = " (pre-release)"
			}
			if f.IsTUI() {
				title := fmt.Sprintf("%s%s — %s  by %s  %s",
					r.TagName, pre, r.Name, r.Author.Login, r.CreatedAt.Format(time.DateOnly))
				body := r.Body
				if body == "" {
					body = "_No release notes provided._"
				}
				return tui.RunPager(title, body, tui.ContentMarkdown)
			}
			fmt.Fprintf(f.IOStreams.Out, "%s%s\n%s\nby %s on %s\n\n%s\n",
				r.TagName, pre, r.Name, r.Author.Login, r.CreatedAt.Format(time.DateOnly), r.Body)
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Release]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newReleaseCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		tagName    string
		name       string
		body       string
		prerelease bool
		target     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a release",
		Long: `Create a new release for the repository.

A tag name is required. If a release name is not provided, the tag
name is used as the release name. Use --prerelease to mark it as a
pre-release.`,
		Example: `  gitee release create --tag v1.0.0 -n "Version 1.0" -b "Release notes..."
  gitee release create --tag v1.0.0-beta1 --prerelease`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			if tagName == "" {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("--tag is required in non-interactive mode")
				}
				if f.IsTUI() {
					if err := releaseCreateForm(&tagName, &name, &body); err != nil {
						return err
					}
				} else {
					tagName, err = cmdutil.AskInput(i18n.T("form.release.tag"), "", false)
					if err != nil {
						return err
					}
					name, err = cmdutil.AskInput(i18n.T("form.release.name"), "", false)
					if err != nil {
						return err
					}
					body, err = cmdutil.OpenEditor(f.IOStreams, "release-body-*.md", "")
					if err != nil {
						return fmt.Errorf("could not open editor: %w", err)
					}
				}
				if tagName == "" {
					return cmdutil.FlagErrorf("tag cannot be empty")
				}
			}
			if name == "" {
				name = tagName
			}
			if target == "" {
				repository, err := client.GetRepo(f.Context, owner, repo)
				if err != nil {
					return fmt.Errorf("failed to resolve default branch: %w", err)
				}
				target = repository.DefaultBranch
				if target == "" {
					return fmt.Errorf("failed to resolve default branch: repository returned an empty default_branch")
				}
			}
			r, err := client.CreateRelease(f.Context, owner, repo, &gitee.CreateReleaseParams{
				TagName:         tagName,
				Name:            name,
				Body:            body,
				Prerelease:      prerelease,
				TargetCommitish: target,
			})
			if err != nil {
				return fmt.Errorf("failed to create release: %w", err)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("release.created", r.TagName, r.Name))
			return nil
		},
	}
	cmd.Flags().StringVar(&tagName, "tag", "", "Tag name")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Release name (default: tag name)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Release description")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Mark as pre-release")
	cmd.Flags().StringVar(&target, "target", "", "Target branch or commit (default: default branch)")
	return cmd
}

func newReleaseDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id-or-tag>",
		Short: "Delete a release by numeric ID or tag name",
		Long:  `Delete a release by its numeric ID or tag name. Requires confirmation unless --yes is provided.`,
		Example: `  gitee release delete 42
  gitee release delete v1.0.0 -y`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := cmdutil.ResolveRepo(cmd)
			if err != nil {
				return err
			}
			displayName := args[0]

			if !yes {
				confirmed, err := cmdutil.ConfirmDestructiveAction(
					f,
					i18n.Tf("release.delete_confirm", displayName),
				)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}

			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			var releaseID int
			if id, err := strconv.Atoi(args[0]); err == nil {
				releaseID = id
			} else {
				r, err := client.GetReleaseByTag(f.Context, owner, repo, args[0])
				if err != nil {
					return fmt.Errorf("release tag %q not found: %w", args[0], err)
				}
				releaseID = r.ID
			}

			if err := client.DeleteRelease(f.Context, owner, repo, releaseID); err != nil {
				return fmt.Errorf("failed to delete release: %w", err)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("release.deleted", displayName))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func releaseListTUI(ctx context.Context, releases []gitee.Release, owner, repo, hostname string, client *gitee.Client) error {
	columns := []table.Column{
		{Title: "ID", Width: 8},
		{Title: "Tag", Width: 18},
		{Title: "Name", Width: 30},
		{Title: "Pre", Width: 4},
		{Title: "Created", Width: 12},
	}

	rows := make([]table.Row, 0, len(releases))
	for _, r := range releases {
		pre := ""
		if r.Prerelease {
			pre = "✓"
		}
		rows = append(rows, table.Row{
			strconv.Itoa(r.ID),
			r.TagName,
			tui.Truncate(r.Name, 28),
			pre,
			r.CreatedAt.Format(time.DateOnly),
		})
	}

	return tui.RunTable(tui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Height:  min(len(releases)+1, 20),
		HelpKeys: []tui.HelpKey{
			{Key: "enter", Desc: "open"},
			{Key: "v", Desc: "preview"},
			{Key: "c", Desc: "copy tag"},
			{Key: "q", Desc: "quit"},
		},
		OnSelect: func(row table.Row) {
			if hostname == "" {
				hostname = config.DefaultHost
			}
			url := fmt.Sprintf("https://%s/%s/%s/releases/tag/%s", hostname, owner, repo, row[1])
			browser.OpenURL(url)
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(row[1])
		},
		OnView: func(row table.Row) tea.Cmd {
			id, err := strconv.Atoi(row[0])
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: fmt.Errorf("invalid release ID: %s", row[0])}
				}
			}
			r, err := client.GetRelease(ctx, owner, repo, id)
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: err}
				}
			}
			pre := ""
			if r.Prerelease {
				pre = " (pre-release)"
			}
			title := fmt.Sprintf("%s%s — %s  by %s  %s",
				r.TagName, pre, r.Name, r.Author.Login, r.CreatedAt.Format(time.DateOnly))
			body := r.Body
			if body == "" {
				body = "_No release notes provided._"
			}
			return tui.NewPagerCmd(title, body, tui.ContentMarkdown)
		},
	})
}

func releaseCreateForm(tagName, name, body *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(i18n.T("form.release.tag")).
				Value(tagName),
			huh.NewInput().
				Title(i18n.T("form.release.name")).
				Description(i18n.T("form.release.name_desc")).
				Value(name),
			huh.NewText().
				Title(i18n.T("form.release.body")).
				Editor(strings.Fields(config.Editor())...).
				Value(body),
		),
	).Run()
}
