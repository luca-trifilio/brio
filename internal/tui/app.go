// Package tui implements the Bubble Tea front-end for brio.
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
	"github.com/charmbracelet/x/ansi"

	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/config"
	"github.com/luca-trifilio/brio/internal/history"
	"github.com/luca-trifilio/brio/internal/hooks"
	"github.com/luca-trifilio/brio/internal/interp"
	"github.com/luca-trifilio/brio/internal/plugins"
	_ "github.com/luca-trifilio/brio/internal/plugins/bruno" // self-registers Bruno loader
	"github.com/luca-trifilio/brio/internal/theme"
	"github.com/luca-trifilio/brio/internal/tui/panes"
	"github.com/luca-trifilio/brio/internal/util"
)

// Model is the root Bubble Tea model.
type Model struct {
	collections []*canonical.Collection
	diagnostics []canonical.Diagnostic
	tree        *panes.TreeModel
	env         *panes.EnvModel
	vars        *panes.VarsModel
	history     panes.HistoryModel

	mode    Mode
	focused Pane
	keymap  KeyMap
	request *panes.RequestModel

	width, height int

	// Active env per collection (key = collection.Path).
	activeEnvs map[string]string

	cmd      textinput.Model
	spinner  spinner.Model
	loading  bool
	statusLn string

	historyStore  *history.Store
	response      *panes.ResponseModel
	help          panes.HelpModel
	pendingYank   bool               // for two-key yc binding
	pendingG      bool               // for two-key gg binding
	activeRequest *canonical.Request // last executed request (may differ from tree selection)
	activeScope   *interp.VarScope

	cfg          *config.Config    // loaded from ~/.config/brio/config.toml
	hookPending  *hookPendingState // set while a credential-refresh hook is in flight
	showSettings bool
	settings     panes.SettingsModel
	picker       *panes.PickerModel
	showPicker   bool
	diagPane     *panes.DiagnosticsModel
	showDiag     bool
	collMgr      *panes.CollectionsModel
	showCollMgr  bool

	// Layout geometry — updated on every View so mouse hit-testing is accurate.
	geom paneGeometry
}

// paneGeometry caches the screen-coordinate boundaries of each pane.
type paneGeometry struct {
	sidebarW  int // total column width of the sidebar (including border)
	treeH     int // total row height of the tree box (including border)
	envH      int // total row height of the env box (including border)
	reqHeight int // total row height of the request box (including border)
}

// NewModel constructs the TUI with one or more loaded collections.
// hookPendingState holds the context needed to retry a request after a
// credential-refresh hook has injected fresh variables.
type hookPendingState struct {
	hook *config.Hook
	c    *canonical.Collection
	env  *canonical.Environment
	req  *canonical.Request
}

// LoadCollections loads collections via the plugin registry, returning all
// successfully-loaded collections plus aggregated diagnostics. A per-path
// fatal error is converted into a diagnostic so partial loads still succeed.
func LoadCollections(entries []config.CollectionEntry) ([]*canonical.Collection, []canonical.Diagnostic) {
	var (
		out   []*canonical.Collection
		diags []canonical.Diagnostic
	)
	for _, e := range entries {
		loader, err := plugins.Resolve(e.Format, e.Path)
		if err != nil {
			diags = append(diags, canonical.Diagnostic{
				Severity: canonical.SeverityError,
				Path:     e.Path,
				Msg:      err.Error(),
			})
			continue
		}
		c, d, err := loader.Load(e.Path)
		diags = append(diags, d...)
		if err != nil {
			diags = append(diags, canonical.Diagnostic{
				Severity: canonical.SeverityError,
				Path:     e.Path,
				Msg:      err.Error(),
			})
			continue
		}
		out = append(out, c)
	}
	return out, diags
}

func NewModel(collections []*canonical.Collection, store *history.Store) *Model {
	return NewModelWithDiagnostics(collections, nil, store)
}

// NewModelWithDiagnostics constructs the TUI with collections plus loader diagnostics.
func NewModelWithDiagnostics(collections []*canonical.Collection, diags []canonical.Diagnostic, store *history.Store) *Model {
	cfg, _ := config.Load() // missing file is fine — returns empty Config
	tree := panes.NewTree(collections)
	ci := textinput.New()
	ci.Prompt = ":"
	ci.CharLimit = 256
	ci.Width = 80

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Yellow)

	var firstColl *canonical.Collection
	if len(collections) > 0 {
		firstColl = collections[0]
	}
	envModel := panes.NewEnv(firstColl)

	m := &Model{
		collections:  collections,
		diagnostics:  diags,
		tree:         tree,
		env:          envModel,
		response:     panes.NewResponse(),
		request:      panes.NewRequest(),
		vars:         panes.NewVars(),
		mode:         ModeNormal,
		focused:      PaneTree,
		keymap:       DefaultKeyMap(),
		activeEnvs:   map[string]string{},
		cmd:          ci,
		spinner:      sp,
		historyStore: store,
		cfg:          cfg,
		settings:     panes.NewSettings(),
		picker:       panes.NewPicker(),
		diagPane:     panes.NewDiagnostics(),
		collMgr:      panes.NewCollections(),
	}
	// Default to the first env, alphabetically, for each collection.
	for _, c := range collections {
		if len(c.Environments) > 0 {
			names := make([]string, 0, len(c.Environments))
			for _, e := range c.Environments {
				names = append(names, e.Name)
			}
			panes.SortEnvNames(names)
			m.activeEnvs[c.Root] = names[0]
		}
	}
	// Apply blocked-methods filter for the initial active env.
	m.syncBlockedMethods()
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
		// Fire the first matching credential-refresh hook.
		// Only attempt once per original request (hookPending == nil guard).
		if m.hookPending == nil {
			tier := theme.ClassifyEnv(msg.Environment)
			if h := hooks.Match(m.cfg.Hooks, msg.Resp, tier); h != nil {
				m.hookPending = &hookPendingState{
					hook: h,
					c:    msg.Collection,
					env:  m.envFor(msg.Collection),
					req:  msg.Request,
				}
				m.loading = true
				m.statusLn = "⚠ hook triggered: " + h.Name + "…"
				return m, tea.Batch(m.spinner.Tick, hooks.Cmd(h))
			}
		}
		m.hookPending = nil // clear retry guard on any other outcome
		r := msg.Resp
		if msg.Resp.Err != nil {
			m.statusLn = "error: " + msg.Resp.Err.Error()
		} else {
			m.statusLn = fmt.Sprintf("%s in %s", msg.Resp.Status, msg.Resp.Elapsed.Round(1e6))
			if len(msg.ExtractedVars) > 0 {
				for k, v := range msg.ExtractedVars {
					m.vars.Set(k, v)
				}
				keys := make([]string, 0, len(msg.ExtractedVars))
				for k := range msg.ExtractedVars {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				m.statusLn += "  ·  set " + strings.Join(keys, ", ")
			}
		}
		m.response.SetResponse(&r, &msg.Resolved, 0, 0) // dimensions filled on next View call
		if m.historyStore != nil {
			_ = m.historyStore.Append(historyEntryFromExecute(msg))
		}
		return m, nil
	case hooks.DoneMsg:
		m.loading = false
		if msg.Err != nil {
			m.hookPending = nil
			m.statusLn = "hook error: " + msg.Err.Error()
			return m, nil
		}
		if m.hookPending != nil {
			// Map output keys to runtime var names as configured in hook.Vars.
			for outKey, varName := range m.hookPending.hook.Vars {
				if val, ok := msg.Vars[outKey]; ok {
					m.vars.Set(varName, val)
				}
			}
			p := m.hookPending
			m.hookPending = nil
			m.statusLn = "hook ok — retrying…"
			return m, tea.Batch(
				m.spinner.Tick,
				runRequestCmd(p.c, p.env, p.req, m.vars.Snapshot()),
			)
		}
		m.hookPending = nil
		return m, nil
	case configEditDoneMsg:
		cfg, err := config.Load()
		if err != nil {
			m.statusLn = "config reload error: " + err.Error()
			return m, nil
		}
		m.cfg = cfg
		m.statusLn = "config reloaded"
		return m, nil
	case errMsg:
		m.statusLn = "error: " + msg.Err.Error()
		return m, nil
	case editorDoneMsg:
		// Reload the collection whose env file was edited.
		for i, c := range m.collections {
			if c.Root == msg.CollectionPath {
				loader, err := plugins.Resolve(c.Format, c.Root)
				if err != nil {
					m.statusLn = "reload error: " + err.Error()
					break
				}
				reloaded, _, err := loader.Load(c.Root)
				if err == nil {
					m.collections[i] = reloaded
					m.syncEnvPane()
				} else {
					m.statusLn = "reload error: " + err.Error()
				}
				break
			}
		}
		return m, nil
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleMouse dispatches mouse events to the appropriate pane.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Modals absorb all mouse input.
	if m.help.Visible || m.history.Visible || m.vars.Visible {
		return m, nil
	}

	x, y := msg.X, msg.Y
	g := m.geom

	// Determine which pane the cursor is over.
	// Row 0 = status bar, rows 1..bodyH = body, last row = cmd line.
	bodyY := y - 1 // 0-indexed within the body region

	var hovered Pane
	switch {
	case x < g.sidebarW && bodyY >= 0 && bodyY < g.treeH:
		hovered = PaneTree
	case x < g.sidebarW && bodyY >= g.treeH:
		hovered = PaneEnv
	case x >= g.sidebarW && bodyY >= 0 && bodyY < g.reqHeight:
		hovered = PaneRequest
	case x >= g.sidebarW && bodyY >= g.reqHeight:
		hovered = PaneResponse
	default:
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.focused = hovered
		switch hovered {
		case PaneTree:
			m.tree.Up()
			m.syncEnvPane()
		case PaneEnv:
			m.env.Up()
		case PaneRequest:
			m.request.HandleKey("k")
		case PaneResponse:
			m.response.HandleKey("k")
		}

	case tea.MouseButtonWheelDown:
		m.focused = hovered
		switch hovered {
		case PaneTree:
			m.tree.Down()
			m.syncEnvPane()
		case PaneEnv:
			m.env.Down()
		case PaneRequest:
			m.request.HandleKey("j")
		case PaneResponse:
			m.response.HandleKey("j")
		}

	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		m.focused = hovered
		switch hovered {
		case PaneTree:
			// box border top (bodyY=0) + title (1) + separator (2) = content starts at bodyY=3
			contentY := bodyY - 3
			if contentY >= 0 {
				m.tree.SetCursor(m.tree.Offset() + contentY)
				m.syncEnvPane()
				// Toggle expand/collapse for collections and folders.
				if sel, ok := m.tree.Selected(); ok && sel.Expandable {
					if m.tree.IsExpanded(sel.Path) {
						m.tree.Collapse()
					} else {
						m.tree.Expand()
					}
					m.syncEnvPane()
				}
			}
		case PaneEnv:
			// box border top (treeH) + title (+1) + separator (+1) = content at treeH+3
			envContentY := bodyY - g.treeH - 3
			if envContentY >= 0 {
				m.env.SetCursor(envContentY)
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string { return m.renderLayout() }

// ----------------------------------------------------------------------------
// Key dispatch
// ----------------------------------------------------------------------------.

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Picker overlay absorbs keys when visible.
	if m.showPicker {
		result, ix, cmd := m.picker.Update(msg)
		switch result {
		case panes.PickerSelected:
			if ix >= 0 && ix < len(m.collections) {
				c := m.collections[ix]
				// Expand the chosen collection in the tree and place cursor on it.
				for _, cc := range m.collections {
					if cc.Root != c.Root {
						m.tree.Expanded[cc.Root] = false
					}
				}
				m.tree.Expanded[c.Root] = true
				m.tree.Rebuild()
				for i, row := range m.tree.Rows() {
					if row.Kind == panes.NodeCollection && row.Path == c.Root {
						m.tree.SetCursor(i)
						break
					}
				}
				m.syncEnvPane()
				m.statusLn = "→ " + c.DisplayName()
			}
			m.showPicker = false
		case panes.PickerCancelled:
			m.showPicker = false
		}
		return m, cmd
	}

	// Collection manager absorbs keys when visible.
	if m.showCollMgr {
		result, cmd := m.collMgr.Update(msg)
		switch result {
		case panes.CollMgrSaved:
			m.cfg.Collections = m.collMgr.Entries()
			if err := config.Save(m.cfg); err != nil {
				m.statusLn = "save error: " + err.Error()
			} else {
				m.statusLn = "collections saved"
			}
			m.showCollMgr = false
			m.collMgr.Close()
			m.reloadCollectionsFromConfig()
		case panes.CollMgrCanceled:
			m.showCollMgr = false
			m.collMgr.Close()
		}
		return m, cmd
	}

	// Diagnostics overlay absorbs keys when visible.
	if m.showDiag {
		switch msg.String() {
		case "esc", "q":
			m.diagPane.Close()
			m.showDiag = false
		case "j", "down":
			m.diagPane.Down()
		case "k", "up":
			m.diagPane.Up()
		case "enter":
			if d, ok := m.diagPane.Selected(); ok && d.Path != "" {
				m.showDiag = false
				m.diagPane.Close()
				return m, openEnvInEditor(d.Path, "")
			}
		}
		return m, nil
	}

	// Settings modal absorbs all keys when visible.
	if m.showSettings {
		// Collection editor sub-mode: delegate all keys to the pane.
		if m.settings.CollEditing() {
			result, cmd := m.settings.UpdateCollEditor(msg)
			switch result {
			case panes.CollEditorSaved:
				paths := m.settings.CollPaths()
				entries := make([]config.CollectionEntry, 0, len(paths))
				for _, p := range paths {
					entries = append(entries, config.CollectionEntry{Path: p})
				}
				m.cfg.Collections = entries
				if err := config.Save(m.cfg); err != nil {
					m.statusLn = "save error: " + err.Error()
				} else {
					m.statusLn = "collections saved"
				}
			case panes.CollEditorCancelled:
				// discard — nothing to do
			}
			return m, cmd
		}

		switch msg.String() {
		case "c":
			paths := make([]string, 0, len(m.cfg.Collections))
			for _, e := range m.cfg.Collections {
				paths = append(paths, e.Path)
			}
			m.settings.EnterCollEditor(paths)
		case "e":
			return m, openConfigInEditor()
		case "r":
			cfg, err := config.Load()
			if err != nil {
				m.statusLn = "config reload error: " + err.Error()
			} else {
				m.cfg = cfg
				m.statusLn = "config reloaded"
			}
		case "q", "esc":
			m.showSettings = false
		}
		return m, nil
	}

	// Help modal absorbs all keys when visible.
	if m.help.Visible {
		switch msg.String() {
		case "esc", "q", "?":
			m.help.Close()
		}
		return m, nil
	}

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

	// ── Two-key gg: jump to top in response / request / tree ─────────────────
	if m.pendingG {
		m.pendingG = false
		switch s {
		case "g":
			switch m.focused {
			case PaneResponse:
				m.response.HandleKey("g")
			case PaneRequest:
				m.request.HandleKey("g")
			case PaneTree:
				m.tree.Top()
				m.syncEnvPane()
			}
		case "s":
			m.showSettings = true
		case "c":
			m.picker.Open(m.collections)
			m.showPicker = true
		case "d":
			m.diagPane.Open(m.diagnostics)
			m.showDiag = true
		}
		// Always consume the second key of the sequence.
		return m, nil
	}

	// ── Two-key yc: copy as curl ──────────────────────────────────────────────
	if m.pendingYank {
		m.pendingYank = false
		if s == "c" {
			return m.copyCurl()
		}
		// Not yc — discard the pending yank and fall through.
	}

	// ── Response pane: forward all vim motions ────────────────────────────────
	if m.focused == PaneResponse {
		// While typing a search or leap query, all keys go to the response model.
		if m.response.Searching() {
			m.response.HandleKey(s)
			return m, nil
		}
		switch s {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		if m.response.HandleKey(s) {
			if yanked := m.response.TakeYanked(); yanked != "" {
				if err := clipboard.WriteAll(yanked); err != nil {
					m.statusLn = "clipboard: " + err.Error()
				} else {
					m.statusLn = fmt.Sprintf("yanked %d lines", strings.Count(yanked, "\n"))
				}
			}
			return m, nil
		}
		// Non-motion keys fall through to the global switch below.
	}

	// ── Request pane: forward vim motions ────────────────────────────────────
	if m.focused == PaneRequest {
		switch s {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		if m.request.HandleKey(s) {
			return m, nil
		}
		// Non-motion keys fall through.
	}

	// ── Global / pane-specific keys ───────────────────────────────────────────
	switch s {
	case "?":
		m.help.Open(m.helpSections())
		return m, nil
	case "I":
		// Open the collection-management modal from any pane.
		m.collMgr.Open(m.cfg.Collections)
		m.showCollMgr = true
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	case ":":
		m.mode = ModeCommand
		m.cmd.Focus()
		m.cmd.SetValue("")
		return m, nil
	case "tab":
		m.focused = nextPane(m.focused)
		m.syncEnvPane()
		return m, nil
	case "shift+tab":
		m.focused = prevPane(m.focused)
		m.syncEnvPane()
		return m, nil

	// Vertical motion — tree / env only; response/request handled above.
	case "j", "down":
		switch m.focused {
		case PaneTree:
			m.tree.Down()
			m.syncEnvPane()
		case PaneEnv:
			m.env.Down()
		}
		return m, nil
	case "k", "up":
		switch m.focused {
		case PaneTree:
			m.tree.Up()
			m.syncEnvPane()
		case PaneEnv:
			m.env.Up()
		}
		return m, nil

	// Half-page scroll — tree only (response/request handled by their HandleKey).
	case "d", "ctrl+d":
		if m.focused == PaneTree {
			m.tree.HalfPageDown()
			m.syncEnvPane()
		}
		return m, nil
	case "u", "ctrl+u":
		if m.focused == PaneTree {
			m.tree.HalfPageUp()
			m.syncEnvPane()
		}
		return m, nil

	// Jump to bottom — tree only; response/request handled by HandleKey.
	case "G":
		if m.focused == PaneTree {
			m.tree.Bottom()
			m.syncEnvPane()
		}
		return m, nil

	// First key of gg — arm pending flag for all scrollable panes.
	case "g":
		m.pendingG = true // arm for gg (jump top) or gs (settings)
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
	case "e":
		if m.focused == PaneEnv {
			return m.openActiveEnvFile()
		}
	case "enter":
		if m.focused == PaneEnv {
			return m.selectEnv()
		}
		return m.executeSelected()
	case "V":
		m.vars.Toggle()
		return m, nil
	case "H":
		return m.openHistory()
	case "y":
		m.pendingYank = true
		return m, nil

	// ── Env cycling: ] = next env, [ = prev env ───────────────────────────────
	case "]":
		m.cycleEnv(+1)
		return m, nil
	case "[":
		m.cycleEnv(-1)
		return m, nil
	}
	return m, nil
}

// cycleEnv steps through the active collection's environments in the given
// direction (+1 = forward, -1 = backward), wrapping around.
func (m *Model) cycleEnv(dir int) {
	c := m.activeCollection()
	if c == nil {
		return
	}
	names := m.env.Names()
	if len(names) == 0 {
		m.statusLn = "no environments in this collection"
		return
	}
	cur := m.activeEnvs[c.Root]
	idx := 0
	for i, n := range names {
		if n == cur {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(names)) % len(names)
	newName := names[idx]
	m.activeEnvs[c.Root] = newName
	m.env.SyncCursor(newName)
	m.syncBlockedMethods()
	m.statusLn = "env \u2192 " + newName
}

// ----------------------------------------------------------------------------
// Commands
// ----------------------------------------------------------------------------.

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
		if c.EnvByName(parts[1]) == nil {
			m.statusLn = "unknown env: " + parts[1]
			return m, nil
		}
		m.activeEnvs[c.Root] = parts[1]
		m.env.SyncCursor(parts[1])
		m.syncBlockedMethods()
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

func (m *Model) openActiveEnvFile() (tea.Model, tea.Cmd) {
	name, ok := m.env.Selected()
	if !ok {
		m.statusLn = "no environment selected"
		return m, nil
	}
	c := m.activeCollection()
	if c == nil {
		m.statusLn = "no collection selected"
		return m, nil
	}
	env := c.EnvByName(name)
	if env == nil {
		m.statusLn = "unknown env: " + name
		return m, nil
	}
	return m, openEnvInEditor(env.Path, c.Root)
}

func (m *Model) selectEnv() (tea.Model, tea.Cmd) {
	name, ok := m.env.Selected()
	if !ok {
		return m, nil
	}
	c := m.activeCollection()
	if c == nil {
		return m, nil
	}
	if c.EnvByName(name) == nil {
		m.statusLn = "unknown env: " + name
		return m, nil
	}
	m.activeEnvs[c.Root] = name
	m.statusLn = "env → " + name
	return m, nil
}

func (m *Model) executeSelected() (tea.Model, tea.Cmd) {
	sel, ok := m.tree.Selected()
	if !ok || sel.Request == nil {
		m.statusLn = "no request selected"
		return m, nil
	}
	if theme.IsMutatingMethod(string(sel.Request.Method)) && theme.MutatingMethodsBlocked(m.activeEnvName()) {
		m.statusLn = "⚠ " + string(sel.Request.Method) + " blocked in production"
		return m, nil
	}
	c := m.collectionFor(sel.CollectionIx)
	env := m.envFor(c)
	m.activeRequest = nil
	m.activeScope = nil
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
	for _, c := range m.collections {
		for _, r := range c.AllRequests() {
			if r.SourcePath == e.RequestPath {
				env := m.envFor(c)
				if e.Environment != "" {
					if envObj := c.EnvByName(e.Environment); envObj != nil {
						env = envObj
					}
				}
				if theme.IsMutatingMethod(string(r.Method)) && theme.MutatingMethodsBlocked(m.activeEnvName()) {
					m.history.Close()
					m.statusLn = "⚠ " + string(r.Method) + " blocked — switch env to replay"
					return m, nil
				}
				_, scope := resolveRequest(c, env, r, m.vars.Snapshot())
				m.activeRequest = r
				m.activeScope = scope
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
// ----------------------------------------------------------------------------.

func (m *Model) helpSections() []panes.HelpSection {
	switch m.focused {
	case PaneTree:
		return append(panes.TreeSections(), panes.GlobalSections()...)
	case PaneEnv:
		return append(panes.EnvSections(), panes.GlobalSections()...)
	case PaneResponse:
		return append(panes.ResponseSections(), panes.GlobalSections()...)
	case PaneRequest:
		return append(panes.RequestSections(), panes.GlobalSections()...)
	default:
		return panes.GlobalSections()
	}
}

// reloadCollectionsFromConfig reloads collections from the current cfg state
// and rebuilds the tree. Diagnostics from loaders replace the previous set.
func (m *Model) reloadCollectionsFromConfig() {
	if m.cfg == nil {
		return
	}
	collections, diags := LoadCollections(m.cfg.Entries())
	m.collections = collections
	m.diagnostics = diags
	m.tree = panes.NewTree(collections)
	if len(collections) > 0 {
		m.env.SetCollection(collections[0])
	} else {
		m.env.SetCollection(nil)
	}
	for _, c := range collections {
		if _, ok := m.activeEnvs[c.Root]; ok {
			continue
		}
		if len(c.Environments) > 0 {
			names := make([]string, 0, len(c.Environments))
			for _, e := range c.Environments {
				names = append(names, e.Name)
			}
			panes.SortEnvNames(names)
			m.activeEnvs[c.Root] = names[0]
		}
	}
	m.syncEnvPane()
}

// OpenCollMgrIfEmpty opens the collection-management modal when no
// collections are configured. Called from the CLI right after the TUI starts
// in the no-collections-loaded path so the user can add one without quitting.
func (m *Model) OpenCollMgrIfEmpty() {
	if len(m.collections) == 0 {
		if m.cfg == nil {
			cfg, _ := config.Load()
			m.cfg = cfg
		}
		m.collMgr.Open(m.cfg.Collections)
		m.showCollMgr = true
	}
}

// syncEnvPane rebuilds the env model for the currently active collection.
// Call whenever the selected collection may have changed.
func (m *Model) syncEnvPane() {
	c := m.activeCollection()
	m.env.SetCollection(c)
	if c != nil {
		m.env.SyncCursor(m.activeEnvs[c.Root])
	}
	m.syncBlockedMethods()
}

// syncBlockedMethods updates the tree's blocked-method filter to match the
// current active environment's safety tier.
func (m *Model) syncBlockedMethods() {
	if theme.MutatingMethodsBlocked(m.activeEnvName()) {
		m.tree.SetBlockedMethods(map[string]bool{"POST": true, "PUT": true, "PATCH": true})
	} else {
		m.tree.SetBlockedMethods(nil)
	}
}

func (m *Model) activeCollection() *canonical.Collection {
	sel, ok := m.tree.Selected()
	if !ok {
		if len(m.collections) > 0 {
			return m.collections[0]
		}
		return nil
	}
	return m.collectionFor(sel.CollectionIx)
}

func (m *Model) collectionFor(ix int) *canonical.Collection {
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
	return m.activeEnvs[c.Root]
}

func (m *Model) envFor(c *canonical.Collection) *canonical.Environment {
	if c == nil {
		return nil
	}
	name := m.activeEnvs[c.Root]
	if name == "" {
		return nil
	}
	return c.EnvByName(name)
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

// activeRequestAndScope returns the request and scope to display in the request
// pane. When a history replay set an explicit active request, that takes
// precedence over the tree selection.
func (m *Model) activeRequestAndScope() (*canonical.Request, *interp.VarScope) {
	if m.activeRequest != nil {
		return m.activeRequest, m.activeScope
	}
	sel, ok := m.tree.Selected()
	if !ok || sel.Request == nil {
		return nil, nil
	}
	c := m.collectionFor(sel.CollectionIx)
	env := m.envFor(c)
	_, scope := resolveRequest(c, env, sel.Request, m.vars.Snapshot())
	return sel.Request, scope
}

// ----------------------------------------------------------------------------
// Status / command line rendering
// ----------------------------------------------------------------------------.

func (m *Model) renderSettings() string {
	return m.settings.View(m.cfg, config.Path(), m.width-8, m.height-8)
}

func (m *Model) renderStatusBar() string {
	c := m.activeCollection()
	collName := ""
	if c != nil {
		collName = theme.StyleText.Render(c.DisplayName())
	}
	if len(m.collections) == 0 && !m.showCollMgr {
		hint := theme.StyleWarning.Render("No collections — press ") +
			theme.StyleActive.Render("I") +
			theme.StyleWarning.Render(" to manage")
		return theme.StyleStatusBar.Width(m.width).Render(hint)
	}
	content := collName + theme.StyleDim.Render(" │ ") + theme.EnvBadge(m.activeEnvName())
	if n := len(m.diagnostics); n > 0 {
		content += theme.StyleDim.Render(" │ ") +
			theme.StyleWarning.Render(fmt.Sprintf("⚠ %d", n))
	}
	return theme.StyleStatusBar.Width(m.width).Render(content)
}

func (m *Model) renderCommandLine() string {
	if m.mode == ModeCommand {
		return ansi.Truncate(m.cmd.View(), m.width, "")
	}

	var modeStyle lipgloss.Style
	var modeLabel string
	switch {
	case m.focused == PaneResponse && m.response.InVisual():
		modeStyle = theme.StyleModeVisual
		modeLabel = "VISUAL"
	case m.mode == ModeInsert:
		modeStyle = theme.StyleModeInsert
		modeLabel = m.mode.String()
	case m.mode == ModeCommand:
		modeStyle = theme.StyleModeCommand
		modeLabel = m.mode.String()
	default:
		modeStyle = theme.StyleModeNormal
		modeLabel = m.mode.String()
	}
	left := modeStyle.Render(modeLabel)
	leftW := lipgloss.Width(left)

	center := m.statusLn
	if m.loading {
		center = m.spinner.View() + " " + center
	}
	centerW := lipgloss.Width(center)

	pendingKey := ""
	if m.pendingYank {
		pendingKey = "y"
	} else if m.pendingG {
		pendingKey = "g"
	}
	var hint string
	if pendingKey != "" {
		hint = theme.StyleActive.Render(pendingKey) + theme.StyleHint.Render("…  ? help")
	} else {
		hint = theme.StyleHint.Render("? help")
	}
	right := hint
	rightW := lipgloss.Width(right)

	remaining := m.width - leftW - rightW
	if remaining < 1 {
		return ansi.Truncate(left, m.width, "")
	}
	// Center the status within the space between mode and help hint.
	padLeft := (remaining - centerW) / 2
	if padLeft < 1 {
		padLeft = 1
	}
	padRight := remaining - centerW - padLeft
	if padRight < 0 {
		padRight = 0
	}
	return left + strings.Repeat(" ", padLeft) + center + strings.Repeat(" ", padRight) + right
}

// ----------------------------------------------------------------------------
// Pane cycling
// ----------------------------------------------------------------------------.

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
