package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all TUI styles.
type Theme struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Header      lipgloss.Style
	SelectedRow lipgloss.Style
	NormalRow   lipgloss.Style
	Listen      lipgloss.Style
	Established lipgloss.Style
	Dim         lipgloss.Style
	StatusBar   lipgloss.Style
	StatusKey   lipgloss.Style
	StatusValue lipgloss.Style
	Help        lipgloss.Style
	Warning     lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	Border      lipgloss.Style
	DetailKey   lipgloss.Style
	DetailValue lipgloss.Style
}

// DefaultTheme returns the default moor color theme.
func DefaultTheme() Theme {
	return Theme{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")),
		SelectedRow: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Bold(true),
		NormalRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")),
		Listen: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),
		Established: lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")),
		Dim: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1),
		StatusKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		StatusValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		DetailKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			Width(16),
		DetailValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
	}
}
