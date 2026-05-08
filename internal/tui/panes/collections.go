package panes

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/config"
	"github.com/luca-trifilio/brio/internal/plugins"
	"github.com/luca-trifilio/brio/internal/theme"
)

// CollMgrResult signals the outcome of a key event handled by the manager.
type CollMgrResult int

// Collection-manager outcome values.
const (
	CollMgrContinue CollMgrResult = iota // still navigating
	CollMgrSaved                         // user saved — caller should persist
	CollMgrCanceled                      // user canceled — caller should close
)

// collMgrStep names the screens of the multi-step modal.
type collMgrStep int

const (
	stepList     collMgrStep = iota // browsing existing collections
	stepPlugin                      // picking a plugin (only shown when >1 registered)
	stepPath                        // entering a path (or autodetect)
	stepCandList                    // multi-select candidates
	stepConfirm                     // final confirm screen
)

// CollectionsModel is the multi-step "manage collections" modal.
//
// Lifecycle:
//
//	NewCollectionsModel() → Open(entries) → Update(key) → CollMgrSaved/Canceled
//
// On CollMgrSaved the caller reads Entries() and persists.
type CollectionsModel struct {
	Visible bool
	step    collMgrStep

	// Working copy of entries (committed only on save).
	entries []config.CollectionEntry
	cursor  int

	// Plugin selection.
	pluginNames []string
	pluginIx    int

	// Path entry.
	pathInput  textinput.Model
	autodetect bool // cursor-on autodetect option vs the input field

	// Candidates discovered after path/autodetect.
	candidates    []string // absolute paths
	candFormat    string   // detected format name
	candSelected  []bool   // parallel to candidates
	candCursor    int
	confirmCursor int // 0 = save, 1 = cancel
	statusMsg     string
}

// NewCollections returns a hidden manager.
func NewCollections() *CollectionsModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 512
	ti.Width = 56
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Yellow)
	return &CollectionsModel{pathInput: ti}
}

// Open prepares the manager with a working copy of entries.
// When no entries exist yet, skips straight to the add flow.
func (m *CollectionsModel) Open(entries []config.CollectionEntry) {
	cp := make([]config.CollectionEntry, len(entries))
	copy(cp, entries)
	m.entries = cp
	m.cursor = 0
	m.statusMsg = ""
	m.Visible = true
	if len(cp) == 0 {
		m.beginAdd()
	} else {
		m.step = stepList
	}
}

// Close hides the manager.
func (m *CollectionsModel) Close() { m.Visible = false }

// Entries returns the current working copy.
func (m *CollectionsModel) Entries() []config.CollectionEntry { return m.entries }

// Update processes a key event.
func (m *CollectionsModel) Update(msg tea.KeyMsg) (CollMgrResult, tea.Cmd) {
	switch m.step {
	case stepList:
		return m.updateList(msg)
	case stepPlugin:
		return m.updatePlugin(msg)
	case stepPath:
		return m.updatePath(msg)
	case stepCandList:
		return m.updateCandList(msg)
	case stepConfirm:
		return m.updateConfirm(msg)
	}
	return CollMgrContinue, nil
}

// ── Step: list ──────────────────────────────────────────────────────────────.

func (m *CollectionsModel) updateList(msg tea.KeyMsg) (CollMgrResult, tea.Cmd) {
	// Cursor position: 0..len(entries)-1 = entries, len(entries) = "Add new".
	maxIx := len(m.entries) // index of "Add new" row

	switch msg.String() {
	case "esc", "q":
		return CollMgrCanceled, nil
	case "j", "down":
		if m.cursor < maxIx {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "d":
		if m.cursor < len(m.entries) {
			m.entries = append(m.entries[:m.cursor], m.entries[m.cursor+1:]...)
			if m.cursor > 0 && m.cursor >= len(m.entries) {
				m.cursor--
			}
			m.statusMsg = "removed"
		}
	case "a", "i":
		m.beginAdd()
	case "enter":
		if m.cursor == maxIx {
			m.beginAdd()
		}
	case "s":
		return CollMgrSaved, nil
	}
	return CollMgrContinue, nil
}

func (m *CollectionsModel) beginAdd() {
	// Collect plugin names.
	names := pluginNames()
	m.pluginNames = names
	m.pluginIx = 0
	if len(names) <= 1 {
		// Skip plugin selection when there's only one (or none).
		m.step = stepPath
	} else {
		m.step = stepPlugin
	}
	m.pathInput.SetValue("")
	m.pathInput.Focus()
	m.autodetect = false
	m.candidates = nil
	m.candSelected = nil
	m.candCursor = 0
	m.statusMsg = ""
}

// ── Step: plugin pick ───────────────────────────────────────────────────────.

func (m *CollectionsModel) updatePlugin(msg tea.KeyMsg) (CollMgrResult, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = stepList
	case "j", "down":
		if m.pluginIx < len(m.pluginNames)-1 {
			m.pluginIx++
		}
	case "k", "up":
		if m.pluginIx > 0 {
			m.pluginIx--
		}
	case "enter":
		m.step = stepPath
	}
	return CollMgrContinue, nil
}

// ── Step: path entry ────────────────────────────────────────────────────────.

func (m *CollectionsModel) updatePath(msg tea.KeyMsg) (CollMgrResult, tea.Cmd) {
	supportsAutodetect := m.selectedPluginSupportsAutodetect()

	// "Tab" moves between input and the autodetect option (if available).
	switch msg.String() {
	case "esc":
		m.pathInput.Blur()
		if len(m.pluginNames) > 1 {
			m.step = stepPlugin
		} else if len(m.entries) == 0 {
			return CollMgrCanceled, nil
		} else {
			m.step = stepList
		}
		return CollMgrContinue, nil
	case "s":
		return CollMgrSaved, nil
	case "tab":
		if supportsAutodetect {
			m.autodetect = !m.autodetect
			if m.autodetect {
				m.pathInput.Blur()
			} else {
				m.pathInput.Focus()
			}
		}
		return CollMgrContinue, nil
	case "enter":
		if m.autodetect {
			m.runAutodetect()
		} else {
			m.runDetect(strings.TrimSpace(m.pathInput.Value()))
		}
		return CollMgrContinue, nil
	}

	if !m.autodetect {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return CollMgrContinue, cmd
	}
	return CollMgrContinue, nil
}

func (m *CollectionsModel) selectedPluginName() string {
	if len(m.pluginNames) == 0 {
		return ""
	}
	if m.pluginIx >= 0 && m.pluginIx < len(m.pluginNames) {
		return m.pluginNames[m.pluginIx]
	}
	return m.pluginNames[0]
}

func (m *CollectionsModel) selectedPluginSupportsAutodetect() bool {
	name := m.selectedPluginName()
	if name == "" {
		return false
	}
	l, err := plugins.Resolve(name, "")
	if err != nil {
		return false
	}
	_, ok := l.(plugins.AutodetectLoader)
	return ok
}

func (m *CollectionsModel) runAutodetect() {
	name := m.selectedPluginName()
	l, err := plugins.Resolve(name, "")
	if err != nil {
		m.statusMsg = "plugin error: " + err.Error()
		return
	}
	ad, ok := l.(plugins.AutodetectLoader)
	if !ok {
		m.statusMsg = "plugin does not support autodetect"
		return
	}
	cands := ad.Autodetect()
	if len(cands) == 0 {
		m.statusMsg = "no collections found"
		return
	}
	m.candidates = cands
	m.candFormat = name
	m.candSelected = make([]bool, len(cands))
	for i := range m.candSelected {
		m.candSelected[i] = true
	}
	m.candCursor = 0
	m.step = stepCandList
}

func (m *CollectionsModel) runDetect(path string) {
	if path == "" {
		m.statusMsg = "enter a path"
		return
	}
	expanded := config.ExpandPath(path)
	cands, fmtName, err := plugins.Default().DetectCollections(expanded)
	if err != nil {
		m.statusMsg = err.Error()
		return
	}
	m.candidates = cands
	m.candFormat = fmtName
	m.candSelected = make([]bool, len(cands))
	for i := range m.candSelected {
		m.candSelected[i] = true
	}
	m.candCursor = 0
	m.step = stepCandList
}

// ── Step: candidate selection ───────────────────────────────────────────────.

func (m *CollectionsModel) updateCandList(msg tea.KeyMsg) (CollMgrResult, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = stepPath
		m.pathInput.Focus()
	case "j", "down":
		if m.candCursor < len(m.candidates)-1 {
			m.candCursor++
		}
	case "k", "up":
		if m.candCursor > 0 {
			m.candCursor--
		}
	case " ", "x":
		if m.candCursor < len(m.candSelected) {
			m.candSelected[m.candCursor] = !m.candSelected[m.candCursor]
		}
	case "a":
		for i := range m.candSelected {
			m.candSelected[i] = true
		}
	case "n":
		for i := range m.candSelected {
			m.candSelected[i] = false
		}
	case "enter":
		m.step = stepConfirm
		m.confirmCursor = 0
	}
	return CollMgrContinue, nil
}

// ── Step: confirm ───────────────────────────────────────────────────────────.

func (m *CollectionsModel) updateConfirm(msg tea.KeyMsg) (CollMgrResult, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = stepCandList
	case "j", "down", "k", "up", "tab":
		m.confirmCursor = 1 - m.confirmCursor
	case "y":
		m.commitCandidates()
		return CollMgrContinue, nil
	case "n":
		m.step = stepList
	case "enter":
		if m.confirmCursor == 0 {
			m.commitCandidates()
		} else {
			m.step = stepList
		}
	}
	return CollMgrContinue, nil
}

func (m *CollectionsModel) commitCandidates() {
	added := 0
	for i, c := range m.candidates {
		if !m.candSelected[i] {
			continue
		}
		// Dedupe by expanded path.
		dup := false
		for _, e := range m.entries {
			if config.ExpandPath(e.Path) == config.ExpandPath(c) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		m.entries = append(m.entries, config.CollectionEntry{Path: c, Format: m.candFormat})
		added++
	}
	if added == 0 {
		m.statusMsg = "nothing added (already configured)"
	} else {
		m.statusMsg = "added"
	}
	m.step = stepList
	m.cursor = len(m.entries)
	if m.cursor > 0 {
		m.cursor--
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────.

func pluginNames() []string {
	r := plugins.Default()
	// DetectAll uses Detect; we just want every registered name. Use a
	// hack: ask DetectAll on a non-existent path and union with a dummy
	// path that always exists. Simpler: introspect via Resolve.
	// We'll just hardcode probing by trying common names via filesystem
	// is fragile. Instead use a small reflective trick: registry has no
	// "ListNames" method, so add one.
	return r.Names()
}

// ── View ────────────────────────────────────────────────────────────────────.

// View renders the modal.
func (m *CollectionsModel) View(width, height int) string {
	if width < 30 {
		width = 30
	}
	innerW := width - 6
	if innerW < 24 {
		innerW = 24
	}

	var body string
	switch m.step {
	case stepList:
		body = m.viewList(innerW)
	case stepPlugin:
		body = m.viewPlugin(innerW)
	case stepPath:
		body = m.viewPath(innerW)
	case stepCandList:
		body = m.viewCandList(innerW)
	case stepConfirm:
		body = m.viewConfirm(innerW)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Mantle).
		Padding(0, 1).
		Width(innerW + 4)

	return box.Render(theme.StyleTitle.Foreground(theme.Mauve).Render("  Collections") + "\n" + body)
}

func (m *CollectionsModel) viewList(innerW int) string {
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)
	addStyle := lipgloss.NewStyle().Foreground(theme.Green).Bold(true)

	var b strings.Builder
	b.WriteString("\n")

	if len(m.entries) == 0 {
		b.WriteString("  " + dimStyle.Render("No collections configured.") + "\n")
	} else {
		for i, e := range m.entries {
			label := e.Path
			if e.Format != "" {
				label += dimStyle.Render("  [" + e.Format + "]")
			}
			row := "   " + valueStyle.Render(label)
			if i == m.cursor {
				row = " " + cursorStyle.Render(" "+stripStyle(label)+" ")
			}
			b.WriteString(row + "\n")
		}
	}

	// "Add new" row.
	addRow := "  " + addStyle.Render("+ Add collection")
	if m.cursor == len(m.entries) {
		addRow = " " + cursorStyle.Render(" + Add collection ")
	}
	b.WriteString(addRow + "\n")

	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	if m.statusMsg != "" {
		b.WriteString("  " + dimStyle.Render(m.statusMsg) + "\n")
	}
	footer := keyStyle.Render("a") + dimStyle.Render(" add  ") +
		keyStyle.Render("d") + dimStyle.Render(" remove  ") +
		keyStyle.Render("s") + dimStyle.Render(" save  ") +
		keyStyle.Render("Esc") + dimStyle.Render(" cancel")
	b.WriteString("  " + footer + "\n")
	return b.String()
}

func (m *CollectionsModel) viewPlugin(innerW int) string {
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render("Pick a format plugin:") + "\n\n")
	for i, n := range m.pluginNames {
		row := "   " + valueStyle.Render(n)
		if i == m.pluginIx {
			row = " " + cursorStyle.Render(" "+n+" ")
		}
		b.WriteString(row + "\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	footer := keyStyle.Render("Enter") + dimStyle.Render(" continue  ") +
		keyStyle.Render("Esc") + dimStyle.Render(" back")
	b.WriteString("  " + footer + "\n")
	return b.String()
}

func (m *CollectionsModel) viewPath(innerW int) string {
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render("Plugin: "+m.selectedPluginName()) + "\n\n")
	b.WriteString("  " + dimStyle.Render("Path:") + "\n")

	inputRow := "  " + m.pathInput.View()
	b.WriteString(inputRow + "\n\n")

	if m.selectedPluginSupportsAutodetect() {
		ad := "  " + dimStyle.Render("[ ] Autodetect from Bruno prefs + CWD")
		if m.autodetect {
			ad = " " + cursorStyle.Render(" [✓] Autodetect from Bruno prefs + CWD ")
		}
		b.WriteString(ad + "\n")
	}

	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	if m.statusMsg != "" {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(theme.Red).Render(m.statusMsg) + "\n")
	}
	footer := keyStyle.Render("Tab") + dimStyle.Render(" toggle  ") +
		keyStyle.Render("Enter") + dimStyle.Render(" detect  ") +
		keyStyle.Render("Esc") + dimStyle.Render(" back")
	b.WriteString("  " + footer + "\n")
	return b.String()
}

func (m *CollectionsModel) viewCandList(innerW int) string {
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)
	checkStyle := lipgloss.NewStyle().Foreground(theme.Green).Bold(true)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render("Found "+pluralize(len(m.candidates), "collection", "collections")+
		" ("+m.candFormat+")") + "\n\n")
	for i, c := range m.candidates {
		mark := "[ ]"
		if m.candSelected[i] {
			mark = checkStyle.Render("[✓]")
		}
		row := "   " + mark + " " + valueStyle.Render(c)
		if i == m.candCursor {
			row = " " + cursorStyle.Render(" "+stripStyle(mark)+" "+c+" ")
		}
		b.WriteString(row + "\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	footer := keyStyle.Render("Space") + dimStyle.Render(" toggle  ") +
		keyStyle.Render("a") + dimStyle.Render(" all  ") +
		keyStyle.Render("n") + dimStyle.Render(" none  ") +
		keyStyle.Render("Enter") + dimStyle.Render(" continue  ") +
		keyStyle.Render("Esc") + dimStyle.Render(" back")
	b.WriteString("  " + footer + "\n")
	return b.String()
}

func (m *CollectionsModel) viewConfirm(innerW int) string {
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render("Will add:") + "\n\n")
	count := 0
	for i, c := range m.candidates {
		if !m.candSelected[i] {
			continue
		}
		b.WriteString("    " + valueStyle.Render(c) + "\n")
		count++
	}
	if count == 0 {
		b.WriteString("    " + dimStyle.Render("(nothing selected)") + "\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render("Format: "+m.candFormat) + "\n\n")

	saveOpt := "  [ Save ]"
	cancelOpt := "  [ Cancel ]"
	if m.confirmCursor == 0 {
		saveOpt = " " + cursorStyle.Render(" [ Save ] ")
	} else {
		cancelOpt = " " + cursorStyle.Render(" [ Cancel ] ")
	}
	b.WriteString(saveOpt + "  " + cancelOpt + "\n\n")

	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	footer := keyStyle.Render("y") + dimStyle.Render(" save  ") +
		keyStyle.Render("n") + dimStyle.Render(" discard  ") +
		keyStyle.Render("Esc") + dimStyle.Render(" back")
	b.WriteString("  " + footer + "\n")
	return b.String()
}

func pluralize(n int, single, plural string) string {
	if n == 1 {
		return "1 " + single
	}
	return itoa(n) + " " + plural
}

func itoa(n int) string {
	// avoid pulling fmt for a single int
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
