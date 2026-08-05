package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const (
	ContentMarkdown = 0
	ContentDiff     = 1
	ContentPlain    = 2
)

type pagerModel struct {
	title     string
	content   string
	ready     bool
	viewport  viewport.Model
	images    []ImageRef
	winWidth  int
	winHeight int
}

func (m pagerModel) Init() tea.Cmd {
	return nil
}

func (m pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "q" || k == "ctrl+c" || k == "esc" {
			return m, tea.Quit
		}
		if msg.String() == "i" && len(m.images) > 0 {
			viewer := newImageViewerCmd(m.images, 0, m.winWidth, m.winHeight)
			return m, tea.Exec(viewer, func(err error) tea.Msg { return nil })
		}
	case tea.WindowSizeMsg:
		m.winWidth = msg.Width
		m.winHeight = msg.Height
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m pagerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m pagerModel) headerView() string {
	title := TitleStyle().Render(m.title)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m pagerModel) footerView() string {
	info := InfoStyle().Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	hint := ""
	if len(m.images) > 0 {
		hint = lipgloss.NewStyle().Foreground(active.HelpKey).Render(fmt.Sprintf(" i: view %d image(s)", len(m.images)))
	}
	lineWidth := max(0, m.viewport.Width-lipgloss.Width(info)-lipgloss.Width(hint))
	line := strings.Repeat("─", lineWidth)
	return lipgloss.JoinHorizontal(lipgloss.Center, line, hint, info)
}

type showPagerMsg struct {
	title   string
	content string
	images  []ImageRef
}

type closePagerMsg struct{}

func NewPagerCmd(title, content string, contentType int) tea.Cmd {
	var images []ImageRef
	if contentType == ContentMarkdown {
		images = ExtractImages(content)
		rendered, err := glamour.Render(content, active.GlamourStyle)
		if err == nil {
			content = rendered
		}
	} else if contentType == ContentDiff {
		content = colorDiff(content)
	}
	return func() tea.Msg {
		return showPagerMsg{title: title, content: content, images: images}
	}
}

func RunPager(title, content string, contentType int) error {
	LoadTheme()
	var images []ImageRef
	if contentType == ContentMarkdown {
		images = ExtractImages(content)
		rendered, err := glamour.Render(content, active.GlamourStyle)
		if err != nil {
			return err
		}
		content = rendered
	} else if contentType == ContentDiff {
		content = colorDiff(content)
	}

	m := pagerModel{
		title:   title,
		content: content,
		images:  images,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

type ViewAction struct {
	Row         table.Row
	ContentType int
	Fetch       bool
	Diff        bool
}

type standaloneExecCmd struct {
	run func() error
}

func (s standaloneExecCmd) SetStdin(r io.Reader)  {}
func (s standaloneExecCmd) SetStdout(w io.Writer) {}
func (s standaloneExecCmd) SetStderr(w io.Writer) {}
func (s standaloneExecCmd) Run() error            { return s.run() }

func colorDiff(diff string) string {
	addStyle := lipgloss.NewStyle().Foreground(ColorGreen)
	delStyle := lipgloss.NewStyle().Foreground(ColorRed)
	hunkStyle := lipgloss.NewStyle().Foreground(ColorBlue)
	headerStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	metaStyle := lipgloss.NewStyle().Foreground(ColorGray)

	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			lines[i] = metaStyle.Render(line)
		case strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index "):
			lines[i] = headerStyle.Render(line)
		case strings.HasPrefix(line, "new file") || strings.HasPrefix(line, "deleted file") || strings.HasPrefix(line, "renamed ") || strings.HasPrefix(line, "status:"):
			lines[i] = metaStyle.Render(line)
		case strings.HasPrefix(line, "@@"):
			lines[i] = hunkStyle.Render(line)
		case strings.HasPrefix(line, "+"):
			lines[i] = addStyle.Render(line)
		case strings.HasPrefix(line, "-"):
			lines[i] = delStyle.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
