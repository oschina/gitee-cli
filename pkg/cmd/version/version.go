package version

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/build"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/tui"
)

const repositoryURL = "https://gitee.com/oschina/gitee-cli"

func NewVersionCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Show the gitee-cli version, commit SHA, build date, Go version, and platform.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if f.IOStreams.ColorEnabled() {
				return printFancy(f)
			}
			return printPlain(f)
		},
	}
}

func printPlain(f *cmdutil.Factory) error {
	w := f.IOStreams.Out
	fmt.Fprintln(w, "Gitee CLI")
	fmt.Fprintln(w, "A command-line tool for Gitee")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %-10s %s\n", "Version", build.Version)
	fmt.Fprintf(w, "  %-10s %s\n", "Commit", build.CommitSHA)
	fmt.Fprintf(w, "  %-10s %s\n", "Built", build.Date)
	fmt.Fprintf(w, "  %-10s %s\n", "Install", build.Distribution)
	fmt.Fprintf(w, "  %-10s %s\n", "Go", runtime.Version())
	fmt.Fprintf(w, "  %-10s %s/%s\n", "Platform", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %-10s %s\n", "Repository", repositoryURL)
	return nil
}

func printFancy(f *cmdutil.Factory) error {
	tui.LoadTheme()
	theme := tui.ActiveTheme()
	w := f.IOStreams.Out

	brand := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Accent).
		Render("GITEE")
	title := lipgloss.JoinHorizontal(lipgloss.Top,
		brand,
		lipgloss.NewStyle().Bold(true).Render(" CLI"),
	)

	subtitle := lipgloss.NewStyle().
		Foreground(theme.HelpDesc).
		Render("A command-line tool for Gitee")

	labelStyle := lipgloss.NewStyle().
		Width(12).
		Foreground(theme.HelpKey).
		Align(lipgloss.Right).
		PaddingRight(2)

	row := func(label string, valueStyle lipgloss.Style, value string) string {
		val := valueStyle.Render(value)
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), val)
	}

	plainValue := lipgloss.NewStyle()
	versionRow := row("Version", lipgloss.NewStyle().Bold(true).Foreground(theme.Success), build.Version)
	commitRow := row("Commit", lipgloss.NewStyle().Foreground(theme.Warning), build.CommitSHA)
	builtRow := row("Built", plainValue, build.Date)
	installRow := row("Install", plainValue, build.Distribution)
	goRow := row("Go", plainValue, runtime.Version())
	platformRow := row("Platform", plainValue, fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))

	separator := lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", 46))

	repositoryRow := row("Repository", lipgloss.NewStyle().Foreground(theme.Accent).Underline(true), repositoryURL)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		subtitle,
		"",
		versionRow,
		commitRow,
		builtRow,
		installRow,
		goRow,
		platformRow,
		"",
		separator,
		repositoryRow,
	)

	box := lipgloss.NewStyle().
		Padding(1, 3).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border)

	fmt.Fprintln(w, box.Render(content))
	return nil
}
