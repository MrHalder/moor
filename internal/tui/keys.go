package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all TUI key bindings.
type KeyMap struct {
	Quit       key.Binding
	Kill       key.Binding
	ForceKill  key.Binding
	Filter     key.Binding
	ClearFilter key.Binding
	Sort       key.Binding
	Refresh    key.Binding
	ToggleAll  key.Binding
	Help       key.Binding
	Detail     key.Binding
	Back       key.Binding
	Up         key.Binding
	Down       key.Binding
	Confirm    key.Binding
	Cancel     key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Kill: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "kill (graceful)"),
		),
		ForceKill: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "force kill"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "cycle sort"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh now"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle all/listen"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Detail: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "j"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "k"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n/esc", "cancel"),
		),
	}
}

// ShortHelp returns key bindings for the compact help view.
func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		km.Quit, km.Kill, km.Filter, km.Sort, km.ToggleAll, km.Detail, km.Help,
	}
}

// FullHelp returns key bindings for the expanded help view.
func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.Up, km.Down, km.Detail, km.Back},
		{km.Kill, km.ForceKill, km.Confirm, km.Cancel},
		{km.Filter, km.ClearFilter, km.Sort, km.ToggleAll},
		{km.Refresh, km.Help, km.Quit},
	}
}
