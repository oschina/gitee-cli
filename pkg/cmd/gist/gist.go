package gist

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
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

func NewGistCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gist",
		Short: "Manage gists",
		Long:  `Create, list, view, and delete gists (code snippets) on Gitee.`,
	}
	cmd.AddCommand(newGistListCmd(f))
	cmd.AddCommand(newGistViewCmd(f))
	cmd.AddCommand(newGistCreateCmd(f))
	cmd.AddCommand(newGistDeleteCmd(f))
	return cmd
}

func newGistListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit      int
		page       int
		jsonFields string
	)
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List your gists",
		Long:    `List your gists with description, file count, and visibility.`,
		Aliases: []string{"ls"},
		Example: `  gitee gist list
  gitee gist list --json=id,description,public,updated_at`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			gists, err := client.ListGists(f.Context, page, limit)
			if err != nil {
				return fmt.Errorf("failed to list gists: %w", err)
			}
			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Gist](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, gists)
				}
				return cmdutil.WriteJSONFields(f.IOStreams.Out, gists, fields)
			}
			if len(gists) == 0 {
				fmt.Fprintln(f.IOStreams.Out, i18n.T("gist.no_results"))
				return nil
			}
			if f.IsTUI() {
				return gistListTUI(f.Context, gists, f.Hostname, client)
			}
			w := tabwriter.NewWriter(f.IOStreams.Out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tDESCRIPTION\tFILES\tVIS\tUPDATED\n")
			for _, g := range gists {
				vis := "secret"
				if g.Public {
					vis = "public"
				}
				desc := tui.Truncate(g.Description, 35)
				id := g.ID
				if len(id) > 8 {
					id = id[:8]
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
					id, desc, len(g.Files), vis, g.UpdatedAt.Format(time.DateOnly))
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of gists per page")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Gist]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func newGistViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View a gist",
		Long:  `View the contents of a gist by its ID, including all files.`,
		Example: `  gitee gist view abc12345
  gitee gist view abc12345 --json=id,description,files,public`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}
			g, err := client.GetGist(f.Context, args[0])
			if err != nil {
				return fmt.Errorf("failed to get gist: %w", err)
			}
			if jsonFields != "" {
				fields, full, listFields := cmdutil.ParseJSONFlag(jsonFields)
				if listFields {
					cmdutil.PrintJSONFieldList[gitee.Gist](f.IOStreams.Out)
					return nil
				}
				if full {
					return cmdutil.WriteJSON(f.IOStreams.Out, g)
				}
				result, err := cmdutil.SelectFields(g, fields)
				if err != nil {
					return err
				}
				return cmdutil.WriteJSON(f.IOStreams.Out, result)
			}

			title := gistTitle(g)
			content := gistContent(g)

			if f.IsTUI() {
				return tui.RunPager(title, content, tui.ContentMarkdown)
			}

			fmt.Fprintf(f.IOStreams.Out, "%s  [%s]  by %s\n%s\n\n",
				g.ID, func() string {
					if g.Public {
						return "public"
					}
					return "secret"
				}(), g.Owner.Login, g.Description)
			for name, file := range g.Files {
				fmt.Fprintf(f.IOStreams.Out, "--- %s ---\n%s\n\n", name, file.Content)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&jsonFields, "json", "j", "", cmdutil.JSONFlagHelp[gitee.Gist]())
	cmd.Flags().Lookup("json").NoOptDefVal = "*"
	return cmd
}

func gistTitle(g *gitee.Gist) string {
	shortID := g.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	vis := "secret"
	if g.Public {
		vis = "public"
	}
	return fmt.Sprintf("%s  [%s]  %s", shortID, vis, g.Description)
}

func gistContent(g *gitee.Gist) string {
	var sb strings.Builder
	for name, file := range g.Files {
		lang := langFromFilename(name)
		sb.WriteString("### ")
		sb.WriteString(name)
		sb.WriteString("\n```")
		sb.WriteString(lang)
		sb.WriteString("\n")
		sb.WriteString(file.Content)
		sb.WriteString("\n```\n\n")
	}
	return sb.String()
}

func newGistCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		description string
		public      bool
		files       []string
	)
	cmd := &cobra.Command{
		Use:   "create [files...]",
		Short: "Create a gist",
		Long: `Create a gist with one or more files.

In interactive mode, you'll be prompted for a description, filename,
and content. In non-interactive mode, provide file paths as arguments
and use --description and --public flags.`,
		Example: `  gitee gist create file1.txt file2.txt
  gitee gist create -d "My snippet" file.py
  gitee gist create -d "Public gist" --public file.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GiteeClient()
			if err != nil {
				return err
			}

			filePaths := append(files, args...)

			if len(filePaths) == 0 {
				if !f.IOStreams.IsStdinTerminal() {
					return cmdutil.FlagErrorf("at least one file is required")
				}
				content := ""
				filename := "gist.txt"
				if f.IsTUI() {
					if err := huh.NewForm(huh.NewGroup(
						huh.NewInput().Title("Description").Value(&description),
						huh.NewInput().Title("Filename").Value(&filename),
						huh.NewText().Title("Content").Editor(strings.Fields(config.Editor())...).Value(&content),
					)).Run(); err != nil {
						return err
					}
				} else {
					description, err = cmdutil.AskInput("Description", "", false)
					if err != nil {
						return err
					}
					filename, err = cmdutil.AskInput("Filename", filename, false)
					if err != nil {
						return err
					}
					content, err = cmdutil.OpenEditor(f.IOStreams, "gist-content-*.txt", "")
					if err != nil {
						return fmt.Errorf("could not open editor: %w", err)
					}
				}
				gistFiles := map[string]gitee.GistFile{filename: {Content: content}}
				g, err := client.CreateGist(f.Context, &gitee.CreateGistParams{
					Description: description,
					Public:      public,
					Files:       gistFiles,
				})
				if err != nil {
					return fmt.Errorf("failed to create gist: %w", err)
				}
				fmt.Fprint(f.IOStreams.Out, i18n.Tf("gist.created", g.ID, g.HTMLURL))
				return nil
			}

			gistFiles := make(map[string]gitee.GistFile, len(filePaths))
			for _, path := range filePaths {
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("failed to read %s: %w", path, err)
				}
				gistFiles[filepath.Base(path)] = gitee.GistFile{Content: string(data)}
			}

			g, err := client.CreateGist(f.Context, &gitee.CreateGistParams{
				Description: description,
				Public:      public,
				Files:       gistFiles,
			})
			if err != nil {
				return fmt.Errorf("failed to create gist: %w", err)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("gist.created", g.ID, g.HTMLURL))
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Gist description")
	cmd.Flags().BoolVar(&public, "public", false, "Make gist public (default: secret)")
	cmd.Flags().StringArrayVarP(&files, "filename", "f", nil, "File to include (can be specified multiple times)")
	return cmd
}

func newGistDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a gist",
		Long:  `Delete a gist by its ID. Requires confirmation unless --yes is provided.`,
		Example: `  gitee gist delete abc12345
  gitee gist delete abc12345 -y`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !yes {
				confirmed, err := cmdutil.ConfirmDestructiveAction(
					f,
					i18n.Tf("gist.delete_confirm", id),
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
			if err := client.DeleteGist(f.Context, id); err != nil {
				return fmt.Errorf("failed to delete gist: %w", err)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("gist.deleted", id))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func gistListTUI(ctx context.Context, gists []gitee.Gist, hostname string, client *gitee.Client) error {
	if hostname == "" {
		hostname = config.DefaultHost
	}
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Description", Width: 35},
		{Title: "Files", Width: 6},
		{Title: "Vis", Width: 7},
		{Title: "Updated", Width: 12},
		{Title: "", Width: 0},
	}
	rows := make([]table.Row, 0, len(gists))
	for _, g := range gists {
		shortID := g.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		vis := "secret"
		if g.Public {
			vis = "public"
		}
		rows = append(rows, table.Row{
			shortID,
			tui.Truncate(g.Description, 33),
			fmt.Sprintf("%d", len(g.Files)),
			vis,
			g.UpdatedAt.Format(time.DateOnly),
			g.ID,
		})
	}
	return tui.RunTable(tui.TableConfig{
		Columns: columns,
		Rows:    rows,
		Height:  min(len(gists)+1, 20),
		HelpKeys: []tui.HelpKey{
			{Key: "enter", Desc: "open"},
			{Key: "v", Desc: "preview"},
			{Key: "c", Desc: "copy id"},
			{Key: "q", Desc: "quit"},
		},
		OnSelect: func(row table.Row) {
			fullID := row[5]
			browser.OpenURL(fmt.Sprintf("https://%s/%s", hostname, fullID))
		},
		OnCopy: func(row table.Row) {
			cmdutil.CopyToClipboard(row[5])
		},
		OnView: func(row table.Row) tea.Cmd {
			fullID := row[5]
			g, err := client.GetGist(ctx, fullID)
			if err != nil {
				return func() tea.Msg {
					return tui.ViewErrorMsg{Err: err}
				}
			}
			return tui.NewPagerCmd(gistTitle(g), gistContent(g), tui.ContentMarkdown)
		},
	})
}

func langFromFilename(name string) string {
	lower := strings.ToLower(name)
	if lower == "dockerfile" {
		return "dockerfile"
	}
	idx := strings.LastIndex(lower, ".")
	if idx < 0 {
		return ""
	}
	ext := lower[idx+1:]
	langs := map[string]string{
		"go": "go", "py": "python", "js": "javascript", "ts": "typescript",
		"tsx": "tsx", "jsx": "jsx", "rs": "rust", "rb": "ruby", "java": "java",
		"c": "c", "cpp": "cpp", "cc": "cpp", "h": "c", "cs": "csharp",
		"php": "php", "sh": "bash", "bash": "bash", "zsh": "bash", "fish": "fish",
		"yaml": "yaml", "yml": "yaml", "json": "json", "toml": "toml",
		"xml": "xml", "html": "html", "css": "css", "scss": "scss",
		"sql": "sql", "md": "markdown", "tf": "hcl", "kt": "kotlin",
		"swift": "swift", "dart": "dart", "lua": "lua", "r": "r",
		"ex": "elixir", "exs": "elixir", "hs": "haskell",
	}
	return langs[ext]
}
