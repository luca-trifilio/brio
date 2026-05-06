// Package tui implements the Bubble Tea front-end for bruno-tui.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/bruno-tui/internal/history"
	"github.com/luca-trifilio/bruno-tui/internal/httpx"
	"github.com/luca-trifilio/bruno-tui/internal/interp"
	"github.com/luca-trifilio/bruno-tui/internal/model"
	"github.com/luca-trifilio/bruno-tui/internal/tui/panes"
	"github.com/luca-trifilio/bruno-tui/internal/util"
)

// Model is the root Bubble Tea model.
type Model struct {
	collections []*model.Collection
	tree        *panes.TreeModel
	vars        *panes.VarsModel
	history     panes.HistoryModel

	mode    Mode
	focused Pane
	keymap  KeyMap

	width, height int

	// Active env per collection (key = collection.Path).
	activeEnvs map[string]string

	cmd      textinput.Model
	spinner  spinner.Model
	loading  bool
	statusLn string

	historyStore *history.Store
	lastResp     *httpx.Response
	pendingYank  bool // for two-key yc binding
}

// NewModel constructs the TUI with one or more loaded collections.
func NewModel(collections []*model.Collection, store *history.Store) *Model {
	tree := panes.NewTree(collections)
	ci := textinput.New()
	ci.Prompt = ":"
	ci.CharLimit = 256
	ci.Width = 80

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	m := &Model{
		collections:  collections,
		tree:         tree,
		vars:         panes.NewVars(),
		mode:         ModeNormal,
		focused:      PaneTree,
		keymap:       DefaultKeyMap(),
		activeEnvs:   map[string]string{},
		cmd:          ci,
		spinner:      sp,
		historyStore: store,
	}
	// Default to the first env, alphabetically, for each collection.
	for _, c := range collections {
		if len(c.Environments) > 0 {
			names := make([]string, 0, len(c.Environments))
			for n := range c.Environments {
				names = append(names, n)
			}
			sort.Strings(names)
			m.activeEnvs[c.Path] = names[0]
		}
	}
	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			return m, cmd
		}
		return m, nil
	case executeMsg:
		m.loading = false
		r := msg.Resp
		m.lastResp = &r
		if msg.Resp.Err != nil {
			m.statusLn = "error: " + msg.Resp.Err.Error()
		} else {
			m.statusLn = fmt.Sprintf("%s in %s", msg.Resp.Status, msg.Resp.Elapsed.Round(1e6))
		}
		if m.historyStore != nil {
			_ = m.historyStore.Append(historyEntryFromExecute(msg))
		}
		return m, nil
	case errMsg:
		m.statusLn = "error: " + msg.Err.Error()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string { return m.renderLayout() }

// ----------------------------------------------------------------------------
// Key dispatch
// ----------------------------------------------------------------------------

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// History modal absorbs keys when visible.
	if m.history.Visible {
		switch msg.String() {
		case "esc", "q":
			m.history.Close()
			return m, nil
		case "j", "down":
			m.history.Down()
			return m, nil
		case "k", "up":
			m.history.Up()
			return m, nil
		case "enter":
			if e, ok := m.history.Selected(); ok {
				return m.replayHistory(e)
			}
			return m, nil
		}
		return m, nil
	}

	// Vars panel takes keys when visible.
	if m.vars.Visible {
		switch msg.String() {
		case "esc":
			if !m.varsEditing() {
				m.vars.Toggle()
				return m, nil
			}
		}
		cmd := m.vars.Update(msg)
		return m, cmd
	}

	switch m.mode {
	case ModeCommand:
		return m.handleCommandKey(msg)
	case ModeNormal:
		return m.handleNormalKey(msg)
	case ModeInsert:
		return m.handleInsertKey(msg)
	}
	return m, nil
}

func (m *Model) varsEditing() bool { return m.vars.Editing() }

func (m *Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
		m.cmd.Blur()
		m.cmd.SetValue("")
		return m, nil
	case "enter":
		input := strings.TrimSpace(m.cmd.Value())
		m.cmd.SetValue("")
		m.cmd.Blur()
		m.mode = ModeNormal
		return m.runCommand(input)
	}
	var cmd tea.Cmd
	m.cmd, cmd = m.cmd.Update(msg)
	return m, cmd
}

func (m *Model) handleInsertKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = ModeNormal
	}
	return m, nil
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	// Two-key yc.
	if m.pendingYank {
		m.pendingYank = false
		if s == "c" {
			return m.copyCurl()
		}
		// Fall through.
	}

	switch s {
	case "ctrl+c":
		return m, tea.Quit
	case ":":
		m.mode = ModeCommand
		m.cmd.Focus()
		m.cmd.SetValue("")
		return m, nil
	case "tab":
		m.focused = nextPane(m.focused)
		return m, nil
	case "shift+tab":
		m.focused = prevPane(m.focused)
		return m, nil
	case "j", "down":
		switch m.focused {
		case PaneTree:
			m.tree.Down()
		}
		return m, nil
	case "k", "up":
		switch m.focused {
		case PaneTree:
			m.tree.Up()
		}
		return m, nil
	case "l", "right":
		if m.focused == PaneTree {
			m.tree.Expand()
		}
		return m, nil
	case "h", "left":
		if m.focused == PaneTree {
			m.tree.Collapse()
		}
		return m, nil
	case "enter":
		return m.executeSelected()
	case "V":
		m.vars.Toggle()
		return m, nil
	case "H":
		return m.openHistory()
	case "y":
		m.pendingYank = true
		return m, nil
	}
	return m, nil
}

// ----------------------------------------------------------------------------
// Commands
// ----------------------------------------------------------------------------

func (m *Model) runCommand(input string) (tea.Model, tea.Cmd) {
	if input == "" {
		return m, nil
	}
	parts := strings.Fields(input)
	switch parts[0] {
	case "q", "quit":
		return m, tea.Quit
	case "env":
		if len(parts) < 2 {
			m.statusLn = "usage: :env <name>"
			return m, nil
		}
		c := m.activeCollection()
		if c == nil {
			m.statusLn = "no collection selected"
			return m, nil
		}
		if _, ok := c.Environments[parts[1]]; !ok {
			m.statusLn = "unknown env: " + parts[1]
			return m, nil
		}
		m.activeEnvs[c.Path] = parts[1]
		m.statusLn = "env → " + parts[1]
		return m, nil
	case "set":
		if len(parts) < 2 {
			m.statusLn = "usage: :set k=v"
			return m, nil
		}
		// Recombine in case the value had spaces (after shell-style splitting).
		rest := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))
		eq := strings.IndexByte(rest, '=')
		if eq < 0 {
			m.statusLn = "usage: :set k=v"
			return m, nil
		}
		k := strings.TrimSpace(rest[:eq])
		v := strings.TrimSpace(rest[eq+1:])
		m.vars.Set(k, v)
		m.statusLn = "set " + k + "=" + v
		return m, nil
	case "history":
		return m.openHistory()
	default:
		m.statusLn = "unknown command: " + parts[0]
	}
	return m, nil
}

func (m *Model) executeSelected() (tea.Model, tea.Cmd) {
	sel, ok := m.tree.Selected()
	if !ok || sel.Request == nil {
		m.statusLn = "no request selected"
		return m, nil
	}
	c := m.collectionFor(sel.CollectionIx)
	env := m.envFor(c)
	m.loading = true
	m.statusLn = "running…"
	cmd := tea.Batch(
		m.spinner.Tick,
		runRequestCmd(c, env, sel.Request, m.vars.Snapshot()),
	)
	return m, cmd
}

func (m *Model) openHistory() (tea.Model, tea.Cmd) {
	if m.historyStore == nil {
		m.statusLn = "no history store"
		return m, nil
	}
	es, err := m.historyStore.Load()
	if err != nil {
		m.statusLn = "history error: " + err.Error()
		return m, nil
	}
	m.history.Open(es)
	return m, nil
}

func (m *Model) replayHistory(e history.Entry) (tea.Model, tea.Cmd) {
	m.history.Close()
	// Find a request matching the saved path.
	for ix, c := range m.collections {
		for _, r := range c.AllRequests() {
			if r.SourcePath == e.RequestPath {
				env := m.envFor(c)
				if e.Environment != "" {
					if envObj, ok := c.Environments[e.Environment]; ok {
						env = envObj
					}
				}
				_ = ix
				m.loading = true
				m.statusLn = "replaying…"
				return m, tea.Batch(
					m.spinner.Tick,
					runRequestCmd(c, env, r, m.vars.Snapshot()),
				)
			}
		}
	}
	m.statusLn = "request not found: " + e.RequestPath
	return m, nil
}

func (m *Model) copyCurl() (tea.Model, tea.Cmd) {
	sel, ok := m.tree.Selected()
	if !ok || sel.Request == nil {
		m.statusLn = "no request selected"
		return m, nil
	}
	c := m.collectionFor(sel.CollectionIx)
	env := m.envFor(c)
	resolved, _ := resolveRequest(c, env, sel.Request, m.vars.Snapshot())
	curl := util.ToCurl(resolved)
	if err := clipboard.WriteAll(curl); err != nil {
		m.statusLn = "clipboard: " + err.Error()
		return m, nil
	}
	m.statusLn = "copied curl to clipboard"
	return m, nil
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func (m *Model) activeCollection() *model.Collection {
	sel, ok := m.tree.Selected()
	if !ok {
		if len(m.collections) > 0 {
			return m.collections[0]
		}
		return nil
	}
	return m.collectionFor(sel.CollectionIx)
}

func (m *Model) collectionFor(ix int) *model.Collection {
	if ix < 0 || ix >= len(m.collections) {
		return nil
	}
	return m.collections[ix]
}

func (m *Model) activeEnvName() string {
	c := m.activeCollection()
	if c == nil {
		return ""
	}
	return m.activeEnvs[c.Path]
}

func (m *Model) envFor(c *model.Collection) *model.Environment {
	if c == nil {
		return nil
	}
	name := m.activeEnvs[c.Path]
	if name == "" {
		return nil
	}
	return c.Environments[name]
}

func (m *Model) scopeForSelected() *interp.VarScope {
	sel, ok := m.tree.Selected()
	if !ok || sel.Request == nil {
		return nil
	}
	c := m.collectionFor(sel.CollectionIx)
	env := m.envFor(c)
	_, scope := resolveRequest(c, env, sel.Request, m.vars.Snapshot())
	return scope
}

// ----------------------------------------------------------------------------
// Status / command line rendering
// ----------------------------------------------------------------------------

func (m *Model) renderStatusBar() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1).
		Width(m.width)
	modeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	c := m.activeCollection()
	collName := ""
	if c != nil {
		collName = c.DisplayName()
	}
	parts := []string{
		modeStyle.Render(m.mode.String()),
		dim.Render(" │ "),
		collName,
		dim.Render(" │ env="),
		m.activeEnvName(),
	}
	return style.Render(strings.Join(parts, ""))
}

func (m *Model) renderCommandLine() string {
	if m.mode == ModeCommand {
		return m.cmd.View()
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	left := m.statusLn
	if m.loading {
		left = m.spinner.View() + " " + left
	}
	help := dim.Render(" [j/k move  l/h fold  Enter run  V vars  H history  yc curl  :q quit]")
	w := m.width - lipgloss.Width(left) - lipgloss.Width(help)
	if w < 1 {
		w = 1
	}
	return left + strings.Repeat(" ", w) + help
}

// ----------------------------------------------------------------------------
// Pane cycling
// ----------------------------------------------------------------------------

func nextPane(p Pane) Pane {
	switch p {
	case PaneTree:
		return PaneRequest
	case PaneRequest:
		return PaneResponse
	case PaneResponse:
		return PaneEnv
	case PaneEnv:
		return PaneTree
	}
	return PaneTree
}

func prevPane(p Pane) Pane {
	switch p {
	case PaneTree:
		return PaneEnv
	case PaneEnv:
		return PaneResponse
	case PaneResponse:
		return PaneRequest
	case PaneRequest:
		return PaneTree
	}
	return PaneTree
}
