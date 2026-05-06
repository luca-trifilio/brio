package tui

// Mode is the current vim-style modal state.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeCommand
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	}
	return "?"
}

// Pane identifies which pane has focus in normal mode.
type Pane int

const (
	PaneTree Pane = iota
	PaneRequest
	PaneResponse
	PaneEnv
	PaneVars
	PaneHistory
)
