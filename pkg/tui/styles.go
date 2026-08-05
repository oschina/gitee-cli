package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	ColorGreen  = lipgloss.Color("#04B575")
	ColorRed    = lipgloss.Color("#FF4672")
	ColorYellow = lipgloss.Color("#FFBF00")
	ColorBlue   = lipgloss.Color("#7DC4E4")
	ColorGray   = lipgloss.Color("244")
	ColorPurple = lipgloss.Color("#8B4789")
)

func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(active.Accent).
		Padding(0, 1)
}

func InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(active.Accent)
}

func successStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(active.Success)
}

func errorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(active.Error)
}

func StateStyle(state string) lipgloss.Style {
	state = strings.ToLower(state)
	switch state {
	case "open":
		return lipgloss.NewStyle().Foreground(active.Success)
	case "closed":
		return lipgloss.NewStyle().Foreground(active.Error)
	case "merged":
		return lipgloss.NewStyle().Foreground(ColorPurple)
	case "rejected":
		return lipgloss.NewStyle().Foreground(active.Error)
	case "progressing":
		return lipgloss.NewStyle().Foreground(active.Warning)
	default:
		return lipgloss.NewStyle().Foreground(active.Border)
	}
}
