package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds bindings used in normal mode.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Quit     key.Binding
	Command  key.Binding
	Vars     key.Binding
	YankCurl key.Binding
	NextPane key.Binding
	PrevPane key.Binding
	Help     key.Binding
	Escape   key.Binding
	History  key.Binding
	NextEnv  key.Binding
	PrevEnv  key.Binding
}

// DefaultKeyMap returns the vim-flavored defaults.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("k", "up")),
		Down:     key.NewBinding(key.WithKeys("j", "down")),
		Left:     key.NewBinding(key.WithKeys("h", "left")),
		Right:    key.NewBinding(key.WithKeys("l", "right")),
		Enter:    key.NewBinding(key.WithKeys("enter")),
		Quit:     key.NewBinding(key.WithKeys("ctrl+c")),
		Command:  key.NewBinding(key.WithKeys(":")),
		Vars:     key.NewBinding(key.WithKeys("V")),
		YankCurl: key.NewBinding(key.WithKeys("y")),
		NextPane: key.NewBinding(key.WithKeys("tab")),
		PrevPane: key.NewBinding(key.WithKeys("shift+tab")),
		Help:     key.NewBinding(key.WithKeys("?")),
		Escape:   key.NewBinding(key.WithKeys("esc")),
		History:  key.NewBinding(key.WithKeys("H")),
		NextEnv:  key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next env")),
		PrevEnv:  key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev env")),
	}
}
