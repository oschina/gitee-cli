package tui

import (
	"errors"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func isUserCancelled(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return true
	}
	s := err.Error()
	return s == "interrupt" || s == "Interrupt" || strings.Contains(s, "user aborted")
}

type FetchResultMsg struct {
	Branch string
	Err    error
}

type EditDoneMsg struct {
	Err error
}

type ViewErrorMsg struct {
	Err error
}

type TableConfig struct {
	Columns  []table.Column
	Rows     []table.Row
	Height   int
	HelpKeys []HelpKey
	OnSelect func(row table.Row)
	OnCopy   func(row table.Row)
	OnView   func(row table.Row) tea.Cmd
	OnDiff   func(row table.Row) tea.Cmd
	OnFetch  func(row table.Row) tea.Cmd
	OnEdit   func(row table.Row) tea.ExecCommand
}

type HelpKey struct {
	Key  string
	Desc string
}

type viewMode int

const (
	modeTable viewMode = iota
	modePager
)

type tableModel struct {
	table      table.Model
	config     TableConfig
	focused    bool
	statusLine string
	mode       viewMode
	pager      pagerModel
	winWidth   int
	winHeight  int
}

func (m tableModel) Init() tea.Cmd {
	return nil
}

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.winWidth = msg.Width
		m.winHeight = msg.Height
		if m.mode == modePager {
			updated, c := m.pager.Update(msg)
			m.pager = updated.(pagerModel)
			return m, c
		}
		return m, nil

	case showPagerMsg:
		m.pager = pagerModel{title: msg.title, content: msg.content, images: msg.images}
		m.mode = modePager
		if m.winWidth > 0 {
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.winWidth, Height: m.winHeight}
			}
		}
		return m, nil

	case closePagerMsg:
		m.mode = modeTable
		return m, nil

	case FetchResultMsg:
		if msg.Err != nil {
			m.statusLine = errorStyle().Render("fetch failed: " + msg.Err.Error())
		} else {
			m.statusLine = successStyle().Render("✓ fetched → " + msg.Branch + "  (git checkout " + msg.Branch + ")")
		}
		return m, nil

	case EditDoneMsg:
		if msg.Err != nil && !isUserCancelled(msg.Err) {
			m.statusLine = errorStyle().Render("edit failed: " + msg.Err.Error())
		} else if msg.Err == nil {
			m.statusLine = successStyle().Render("✓ saved")
		}
		return m, nil

	case ViewErrorMsg:
		if msg.Err != nil {
			m.statusLine = errorStyle().Render("preview failed: " + msg.Err.Error())
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == modePager {
			if k := msg.String(); k == "q" || k == "esc" || k == "ctrl+c" {
				m.mode = modeTable
				return m, nil
			}
			updated, c := m.pager.Update(msg)
			m.pager = updated.(pagerModel)
			return m, c
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.config.OnSelect != nil {
				m.config.OnSelect(m.table.SelectedRow())
			}
		case "c":
			if m.config.OnCopy != nil {
				m.config.OnCopy(m.table.SelectedRow())
			}
		case "v":
			if m.config.OnView != nil {
				return m, m.config.OnView(m.table.SelectedRow())
			}
		case "d":
			if m.config.OnDiff != nil {
				return m, m.config.OnDiff(m.table.SelectedRow())
			}
		case "f":
			if m.config.OnFetch != nil {
				m.statusLine = InfoStyle().Render("fetching…")
				return m, m.config.OnFetch(m.table.SelectedRow())
			}
		case "e":
			if m.config.OnEdit != nil {
				execCmd := m.config.OnEdit(m.table.SelectedRow())
				if execCmd != nil {
					return m, tea.Exec(execCmd, func(err error) tea.Msg {
						return EditDoneMsg{Err: err}
					})
				}
			}
		case "esc":
			if m.focused {
				m.table.Blur()
				m.focused = false
			} else {
				m.table.Focus()
				m.focused = true
			}
			return m, nil
		default:
			m.table, cmd = m.table.Update(msg)
		}

	default:
		if m.mode == modePager {
			updated, c := m.pager.Update(msg)
			m.pager = updated.(pagerModel)
			return m, c
		}
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

func (m tableModel) View() string {
	if m.mode == modePager {
		return m.pager.View()
	}

	base := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(active.Border).
		Render(m.table.View()) + "\n"

	if m.statusLine != "" {
		return base + m.statusLine + "\n"
	}

	if len(m.config.HelpKeys) > 0 {
		return base + m.helpView()
	}
	return base
}

func (m tableModel) helpView() string {
	keyStyle := lipgloss.NewStyle().Foreground(active.HelpKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(active.HelpDesc)
	sep := descStyle.Render(" · ")

	parts := make([]string, 0, len(m.config.HelpKeys))
	for _, h := range m.config.HelpKeys {
		parts = append(parts, keyStyle.Render(h.Key)+" "+descStyle.Render(h.Desc))
	}

	line := ""
	for i, p := range parts {
		if i > 0 {
			line += sep
		}
		line += p
	}
	return line + "\n"
}

func RunTable(cfg TableConfig) error {
	LoadTheme()
	height := cfg.Height
	if height == 0 {
		height = 15
	}

	t := table.New(
		table.WithColumns(cfg.Columns),
		table.WithRows(cfg.Rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(active.Border).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(active.SelectedFg).
		Background(active.SelectedBg).
		Bold(false)
	t.SetStyles(s)

	m := tableModel{
		table:   t,
		config:  cfg,
		focused: true,
		mode:    modeTable,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type HuhExecCmd struct {
	run func() error
}

func NewHuhExecCmd(fn func() error) tea.ExecCommand {
	return &HuhExecCmd{run: fn}
}

func (h *HuhExecCmd) SetStdin(r io.Reader)  {}
func (h *HuhExecCmd) SetStdout(w io.Writer) {}
func (h *HuhExecCmd) SetStderr(w io.Writer) {}
func (h *HuhExecCmd) Run() error            { return h.run() }
