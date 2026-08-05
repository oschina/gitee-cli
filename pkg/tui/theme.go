package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	cfg "gitee.com/oschina/gitee-cli/internal/config"
)

type Theme struct {
	Accent       lipgloss.Color
	Border       lipgloss.Color
	HeaderFg     lipgloss.Color
	SelectedFg   lipgloss.Color
	SelectedBg   lipgloss.Color
	HelpKey      lipgloss.Color
	HelpDesc     lipgloss.Color
	Success      lipgloss.Color
	Error        lipgloss.Color
	Warning      lipgloss.Color
	GlamourStyle string
}

var themes = map[string]Theme{
	"dark": {
		Accent:       lipgloss.Color("#04B575"),
		Border:       lipgloss.Color("240"),
		HeaderFg:     lipgloss.Color("252"),
		SelectedFg:   lipgloss.Color("230"),
		SelectedBg:   lipgloss.Color("238"),
		HelpKey:      lipgloss.Color("246"),
		HelpDesc:     lipgloss.Color("243"),
		Success:      lipgloss.Color("#04B575"),
		Error:        lipgloss.Color("#FF4672"),
		Warning:      lipgloss.Color("#FFBF00"),
		GlamourStyle: "dark",
	},
	"light": {
		Accent:       lipgloss.Color("#0062CC"),
		Border:       lipgloss.Color("250"),
		HeaderFg:     lipgloss.Color("238"),
		SelectedFg:   lipgloss.Color("232"),
		SelectedBg:   lipgloss.Color("254"),
		HelpKey:      lipgloss.Color("242"),
		HelpDesc:     lipgloss.Color("246"),
		Success:      lipgloss.Color("#2DA44E"),
		Error:        lipgloss.Color("#CF222E"),
		Warning:      lipgloss.Color("#9A6700"),
		GlamourStyle: "light",
	},
	"dracula": {
		Accent:       lipgloss.Color("#BD93F9"),
		Border:       lipgloss.Color("#6272A4"),
		HeaderFg:     lipgloss.Color("#F8F8F2"),
		SelectedFg:   lipgloss.Color("#F8F8F2"),
		SelectedBg:   lipgloss.Color("#44475A"),
		HelpKey:      lipgloss.Color("#BD93F9"),
		HelpDesc:     lipgloss.Color("#6272A4"),
		Success:      lipgloss.Color("#50FA7B"),
		Error:        lipgloss.Color("#FF5555"),
		Warning:      lipgloss.Color("#FFB86C"),
		GlamourStyle: "dracula",
	},
	"tokyo-night": {
		Accent:       lipgloss.Color("#7AA2F7"),
		Border:       lipgloss.Color("#3B4261"),
		HeaderFg:     lipgloss.Color("#C0CAF5"),
		SelectedFg:   lipgloss.Color("#C0CAF5"),
		SelectedBg:   lipgloss.Color("#283457"),
		HelpKey:      lipgloss.Color("#7AA2F7"),
		HelpDesc:     lipgloss.Color("#565F89"),
		Success:      lipgloss.Color("#9ECE6A"),
		Error:        lipgloss.Color("#F7768E"),
		Warning:      lipgloss.Color("#E0AF68"),
		GlamourStyle: "tokyo-night",
	},
	"pink": {
		Accent:       lipgloss.Color("#FF79C6"),
		Border:       lipgloss.Color("#FF79C6"),
		HeaderFg:     lipgloss.Color("#F8F8F2"),
		SelectedFg:   lipgloss.Color("#F8F8F2"),
		SelectedBg:   lipgloss.Color("#44475A"),
		HelpKey:      lipgloss.Color("#FF79C6"),
		HelpDesc:     lipgloss.Color("#BD93F9"),
		Success:      lipgloss.Color("#50FA7B"),
		Error:        lipgloss.Color("#FF5555"),
		Warning:      lipgloss.Color("#FFB86C"),
		GlamourStyle: "pink",
	},
}

var active Theme

func init() {
	LoadTheme()
}

func LoadTheme() {
	name := cfg.Get(cfg.KeyTheme)
	if name == "" || name == "auto" {
		if termenv.HasDarkBackground() {
			active = themes["dark"]
		} else {
			active = themes["light"]
		}
		active.GlamourStyle = "auto"
		return
	}
	t, ok := themes[name]
	if !ok {
		if termenv.HasDarkBackground() {
			active = themes["dark"]
		} else {
			active = themes["light"]
		}
		active.GlamourStyle = "auto"
		return
	}
	active = t
}

func ActiveTheme() Theme {
	return active
}
