package panes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/httpx"
	"github.com/luca-trifilio/brio/internal/theme"
)

const leapLabels = "asdfjklghqwertyuiopzxcvbnmASDFJKLGHQWERTYUIOPZXCVBNM"

// leapTarget is one labeled jump position within the visible viewport.
type leapTarget struct {
	absLine int    // absolute line index in r.lines
	col     int    // byte offset of the match in the plain text of that line
	label   string // single character label shown at that position
}

// ResponseModel holds rendered lines and vim scroll/search state.
type ResponseModel struct {
	resp     *httpx.Response
	resolved *httpx.ResolvedRequest // the request that produced resp (for sent-body display)
	lines    []string               // all rendered lines
	contentW int                    // exact display-content width (scrollbar + gutter excluded), set by rebuild()
	offset   int                    // first visible line index
	cursor   int                    // absolute index of the highlighted line
	height   int                    // last known viewport height
	width    int                    // last known viewport width
	count    int                    // pending numeric prefix (e.g. 5 in "5j")

	// /? search state
	searchBuf     string
	searchQuery   string
	searchForward bool
	searching     bool
	matches       []int
	matchIdx      int

	// s leap state
	leaping     bool         // true while in leap mode (waiting for label or first char)
	leapChar    string       // the trigger character typed after s
	leapTargets []leapTarget // labeled targets in current viewport

	// v/V visual (linewise) selection state
	visual       bool   // true while in visual mode
	visualAnchor int    // line where visual mode was entered (fixed end of selection)
	yanked       string // text copied by y in visual mode; root model reads and clears via TakeYanked
}

func NewResponse() *ResponseModel { return &ResponseModel{searchForward: true} }

// InVisual reports whether visual linewise selection is active.
func (r *ResponseModel) InVisual() bool { return r.visual }

// TakeYanked returns and clears the last text yanked in visual mode.
// Called by the root model after HandleKey to write to the clipboard.
func (r *ResponseModel) TakeYanked() string {
	s := r.yanked
	r.yanked = ""
	return s
}

// VisualLineCount returns the number of selected lines (0 when not in visual mode).
func (r *ResponseModel) VisualLineCount() int {
	if !r.visual {
		return 0
	}
	start, end := r.visualAnchor, r.cursor
	if start > end {
		start, end = end, start
	}
	return end - start + 1
}

// selectedText returns the plain-text content of the visual selection.
func (r *ResponseModel) selectedText() string {
	start, end := r.visualAnchor, r.cursor
	if start > end {
		start, end = end, start
	}
	var sb strings.Builder
	for i := start; i <= end && i < len(r.lines); i++ {
		sb.WriteString(stripStyle(r.lines[i]))
		sb.WriteString("\n")
	}
	return sb.String()
}

// SetResponse stores a new response and resets scroll to top.
// resolved is the interpolated request that was sent; pass nil if unavailable.
func (r *ResponseModel) SetResponse(resp *httpx.Response, resolved *httpx.ResolvedRequest, width, height int) {
	r.resp = resp
	r.resolved = resolved
	r.width = width
	r.height = height
	r.offset = 0
	r.cursor = 0
	r.count = 0
	r.clearSearch()
	r.leaping = false
	r.leapTargets = nil
	r.rebuild()
}

// Resize updates dimensions and clamps scroll.
func (r *ResponseModel) Resize(width, height int) {
	r.width = width
	r.height = height
	r.rebuild()
	r.clamp()
}

// Searching returns true while the user is in search or leap input mode.
func (r *ResponseModel) Searching() bool { return r.searching || r.leaping }

func (r *ResponseModel) rebuild() {
	if r.width <= 0 {
		return
	}
	// Two-pass: first pass gives us a line count → gutterW → exact contentW;
	// second pass wraps at contentW so pad() in View never has to truncate.
	pre := buildLines(r.resp, r.resolved, r.width)
	nLines := len(pre)
	if nLines < 1 {
		nLines = 1
	}
	gutterW := len(fmt.Sprintf("%d", nLines)) + 1 // digit-width + 1 separator space
	r.contentW = r.width - 1 - gutterW            // -1 scrollbar column
	if r.contentW < 20 {
		r.contentW = 20
	}
	r.lines = buildLines(r.resp, r.resolved, r.contentW)
	r.recomputeMatches()
}

func (r *ResponseModel) clearSearch() {
	r.searchBuf = ""
	r.searchQuery = ""
	r.searching = false
	r.matches = nil
	r.matchIdx = 0
}

func (r *ResponseModel) recomputeMatches() {
	r.matches = nil
	if r.searchQuery == "" {
		return
	}
	q := strings.ToLower(r.searchQuery)
	for i, l := range r.lines {
		if strings.Contains(strings.ToLower(stripStyle(l)), q) {
			r.matches = append(r.matches, i)
		}
	}
}

func (r *ResponseModel) jumpToMatch(forward bool) {
	if len(r.matches) == 0 {
		return
	}
	if forward {
		for i, m := range r.matches {
			if m > r.cursor {
				r.matchIdx = i
				r.cursor = m
				r.clamp()
				return
			}
		}
		r.matchIdx = 0
		r.cursor = r.matches[0]
	} else {
		for i := len(r.matches) - 1; i >= 0; i-- {
			if r.matches[i] < r.cursor {
				r.matchIdx = i
				r.cursor = r.matches[i]
				r.clamp()
				return
			}
		}
		r.matchIdx = len(r.matches) - 1
		r.cursor = r.matches[r.matchIdx]
	}
	r.clamp()
}

// buildLeapTargets scans the visible viewport for all occurrences of ch and
// assigns a label to each, up to len(leapLabels) matches.
func (r *ResponseModel) buildLeapTargets(ch string) {
	r.leapTargets = nil
	viewH := r.viewH()
	end := r.offset + viewH
	if end > len(r.lines) {
		end = len(r.lines)
	}
	labelIdx := 0
	for absLine := r.offset; absLine < end; absLine++ {
		plain := stripStyle(r.lines[absLine])
		lower := strings.ToLower(plain)
		search := strings.ToLower(ch)
		idx := 0
		for {
			pos := strings.Index(lower[idx:], search)
			if pos < 0 {
				break
			}
			col := idx + pos
			if labelIdx >= len(leapLabels) {
				break
			}
			r.leapTargets = append(r.leapTargets, leapTarget{
				absLine: absLine,
				col:     col,
				label:   string(leapLabels[labelIdx]),
			})
			labelIdx++
			idx = col + len(search)
			if idx >= len(lower) {
				break
			}
		}
		if labelIdx >= len(leapLabels) {
			break
		}
	}
}

func (r *ResponseModel) viewH() int {
	h := r.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (r *ResponseModel) clamp() {
	max := len(r.lines) - r.viewH()
	if max < 0 {
		max = 0
	}
	if r.offset > max {
		r.offset = max
	}
	if r.offset < 0 {
		r.offset = 0
	}
	if r.cursor >= len(r.lines) && len(r.lines) > 0 {
		r.cursor = len(r.lines) - 1
	}
	if r.cursor < 0 {
		r.cursor = 0
	}
	if r.cursor < r.offset {
		r.offset = r.cursor
	}
	if r.cursor >= r.offset+r.viewH() {
		r.offset = r.cursor - r.viewH() + 1
	}
}

func (r *ResponseModel) n() int {
	if r.count <= 0 {
		return 1
	}
	return r.count
}

// HandleKey processes a vim key. Returns true if the key was consumed.
func (r *ResponseModel) HandleKey(key string) bool {
	// Visual linewise selection mode.
	if r.visual {
		switch key {
		case "esc", "v", "V":
			r.visual = false
			return true
		case "y":
			r.yanked = r.selectedText()
			r.visual = false
			return true
		// All movement keys extend the selection by moving cursor (anchor stays fixed).
		case "j", "down":
			n := r.n()
			r.count = 0
			r.cursor += n
			r.clamp()
			return true
		case "k", "up":
			n := r.n()
			r.count = 0
			r.cursor -= n
			r.clamp()
			return true
		case "d", "ctrl+d":
			r.cursor += r.halfPage()
			r.clamp()
			return true
		case "u", "ctrl+u":
			r.cursor -= r.halfPage()
			r.clamp()
			return true
		case "f", "ctrl+f", "pgdown":
			r.cursor += r.viewH()
			r.clamp()
			return true
		case "b", "ctrl+b", "pgup":
			r.cursor -= r.viewH()
			r.clamp()
			return true
		case "g":
			r.cursor = 0
			r.offset = 0
			return true
		case "G":
			n := r.n()
			r.count = 0
			if n > 1 {
				r.cursor = n - 1
			} else {
				r.cursor = len(r.lines) - 1
			}
			r.clamp()
			return true
		default:
			// Numeric prefix accumulation still works in visual mode.
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				r.count = r.count*10 + int(key[0]-'0')
				return true
			}
			if key == "0" && r.count > 0 {
				r.count = r.count * 10
				return true
			}
			// Any other key exits visual and falls through.
			r.visual = false
			return false
		}
	}

	// Leap mode: accumulate characters, show labeled targets, jump on label press.
	if r.leaping {
		switch key {
		case "esc":
			r.leaping = false
			r.leapTargets = nil
			r.leapChar = ""
			return true
		case "backspace", "ctrl+h":
			if len(r.leapChar) > 0 {
				_, size := utf8.DecodeLastRuneInString(r.leapChar)
				r.leapChar = r.leapChar[:len(r.leapChar)-size]
				r.buildLeapTargets(r.leapChar)
			}
			return true
		case "enter":
			// Jump to first target if any.
			if len(r.leapTargets) > 0 {
				r.cursor = r.leapTargets[0].absLine
				r.clamp()
			}
			r.leaping = false
			r.leapTargets = nil
			r.leapChar = ""
			return true
		}
		if len(key) != 1 {
			return true
		}
		// If targets are showing, check if key is a label — jump and exit.
		if len(r.leapTargets) > 0 {
			for _, t := range r.leapTargets {
				if t.label == key {
					r.cursor = t.absLine
					r.clamp()
					r.leaping = false
					r.leapTargets = nil
					r.leapChar = ""
					return true
				}
			}
		}
		// Otherwise append to the search string and recompute.
		r.leapChar += key
		r.buildLeapTargets(r.leapChar)
		if len(r.leapTargets) == 0 {
			r.leaping = false
			r.leapChar = ""
		} else if len(r.leapTargets) == 1 {
			// Only one match — jump immediately.
			r.cursor = r.leapTargets[0].absLine
			r.clamp()
			r.leaping = false
			r.leapTargets = nil
			r.leapChar = ""
		}
		return true
	}

	// /? search input mode.
	if r.searching {
		switch key {
		case "enter":
			r.searchQuery = r.searchBuf
			r.searching = false
			r.recomputeMatches()
			r.jumpToMatch(r.searchForward)
		case "esc":
			r.searching = false
			r.searchBuf = ""
		case "backspace", "ctrl+h":
			if len(r.searchBuf) > 0 {
				r.searchBuf = r.searchBuf[:len(r.searchBuf)-1]
			}
		default:
			if len(key) == 1 {
				r.searchBuf += key
			}
		}
		return true
	}

	// Accumulate numeric prefix.
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		r.count = r.count*10 + int(key[0]-'0')
		return true
	}
	if key == "0" && r.count == 0 {
		r.cursor = 0
		r.offset = 0
		return true
	}
	if key == "0" {
		r.count = r.count * 10
		return true
	}

	n := r.n()
	r.count = 0

	switch key {
	case "j", "down":
		r.cursor += n
		r.clamp()
	case "k", "up":
		r.cursor -= n
		r.clamp()
	case "d", "ctrl+d":
		r.cursor += r.halfPage()
		r.clamp()
	case "u", "ctrl+u":
		r.cursor -= r.halfPage()
		r.clamp()
	case "f", "ctrl+f", "pgdown":
		r.cursor += r.viewH()
		r.clamp()
	case "b", "ctrl+b", "pgup":
		r.cursor -= r.viewH()
		r.clamp()
	case "g":
		r.cursor = 0
		r.offset = 0
	case "G":
		if n > 1 {
			r.cursor = n - 1
		} else {
			r.cursor = len(r.lines) - 1
		}
		r.clamp()
	case "v", "V":
		r.visual = true
		r.visualAnchor = r.cursor
	case "s":
		r.leaping = true
		r.leapChar = ""
		r.leapTargets = nil
	case "/":
		r.searching = true
		r.searchForward = true
		r.searchBuf = ""
	case "?":
		r.searching = true
		r.searchForward = false
		r.searchBuf = ""
	case "n":
		r.jumpToMatch(r.searchForward)
	case "N":
		r.jumpToMatch(!r.searchForward)
	default:
		return false
	}
	return true
}

func (r *ResponseModel) halfPage() int {
	h := r.viewH() / 2
	if h < 1 {
		h = 1
	}
	return h
}

// View renders the visible portion of the response.
func (r *ResponseModel) View(width, height int, focused bool) string {
	if width != r.width || height != r.height {
		r.Resize(width, height)
	}

	title := "Response"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}
	separator := theme.StyleDim.Render(strings.Repeat("─", width))
	header := theme.StyleTitle.Render(title) + "\n" + separator

	viewH := r.viewH()

	if len(r.lines) == 0 {
		return header + "\n" + theme.StyleDim.Render("  (no response yet — press Enter on a request)")
	}

	end := r.offset + viewH
	if end > len(r.lines) {
		end = len(r.lines)
	}
	visible := make([]string, viewH)
	copy(visible, r.lines[r.offset:end])

	scrollbar := buildScrollbar(r.offset, len(r.lines), viewH)

	pct := 0
	scrollable := len(r.lines) > viewH
	if scrollable {
		pct = r.offset * 100 / (len(r.lines) - viewH)
		if pct > 100 {
			pct = 100
		}
	} else {
		pct = 100
	}

	// Gutter: wide enough for the largest line number + 1 separator space.
	// gutterW is recomputed from the final line count (may differ from rebuild's
	// preliminary count by at most 1 digit, which is harmless).
	gutterW := len(fmt.Sprintf("%d", len(r.lines))) + 1
	contentW := r.contentW
	if contentW < 1 {
		contentW = 1
	}

	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Sky)
	gutterCursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Lavender).Bold(true)
	gutterDimStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	matchStyle := lipgloss.NewStyle().Background(theme.Surface0).Foreground(theme.Yellow)
	activeMatchStyle := lipgloss.NewStyle().Background(theme.Yellow).Foreground(theme.Base)
	leapDimStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	leapLabelStyle := lipgloss.NewStyle().Background(theme.Mauve).Foreground(theme.Base).Bold(true)

	// Visual selection bounds (inclusive).
	visStart, visEnd := -1, -1
	if r.visual {
		visStart, visEnd = r.visualAnchor, r.cursor
		if visStart > visEnd {
			visStart, visEnd = visEnd, visStart
		}
	}

	// Build a lookup: absLine → []leapTarget for O(1) access in the render loop.
	leapByLine := map[int][]leapTarget{}
	if r.leaping && r.leapChar != "" {
		for _, t := range r.leapTargets {
			leapByLine[t.absLine] = append(leapByLine[t.absLine], t)
		}
	}

	rows := make([]string, viewH)
	for i, line := range visible {
		bar := " "
		if scrollable && i < len(scrollbar) {
			bar = scrollbar[i]
		}
		absLine := r.offset + i

		// Relative line number gutter.
		var gutterStr string
		if absLine == r.cursor {
			n := fmt.Sprintf("%*d ", gutterW-1, absLine+1)
			gutterStr = gutterCursorStyle.Render(n)
		} else {
			rel := absLine - r.cursor
			if rel < 0 {
				rel = -rel
			}
			n := fmt.Sprintf("%*d ", gutterW-1, rel)
			gutterStr = gutterDimStyle.Render(n)
		}

		cell := pad(line, contentW)

		switch {
		case r.leaping && r.leapChar != "":
			targets, hasTargets := leapByLine[absLine]
			if hasTargets {
				cell = renderLeapLine(stripStyle(cell), targets, leapDimStyle, leapLabelStyle, contentW)
			} else {
				cell = leapDimStyle.Render(pad(stripStyle(cell), contentW))
			}
		case r.visual && absLine == r.cursor:
			// Cursor end of the selection — distinct mauve tint so user can see where the cursor is.
			cell = theme.StyleVisualCursor.Render(pad(stripStyle(cell), contentW))
		case r.visual && absLine >= visStart && absLine <= visEnd:
			cell = theme.StyleVisualLine.Render(pad(stripStyle(cell), contentW))
		case focused && absLine == r.cursor:
			// cell is already padded to contentW; render the full width as cursor line.
			cell = cursorStyle.Render(cell)
		case r.searchQuery != "" && isMatch(r.lines, absLine, r.searchQuery):
			isCurrent := len(r.matches) > 0 && r.matchIdx < len(r.matches) && r.matches[r.matchIdx] == absLine
			if isCurrent {
				cell = highlightMatch(cell, r.searchQuery, activeMatchStyle, contentW)
			} else {
				cell = highlightMatch(cell, r.searchQuery, matchStyle, contentW)
			}
		}

		rows[i] = gutterStr + cell + bar
	}

	var bottomLine string
	switch {
	case r.visual:
		nSel := r.VisualLineCount()
		bottomLine = theme.StyleModeVisual.Render("-- VISUAL --") +
			theme.StyleDim.Render(fmt.Sprintf("  %d line", nSel))
		if nSel != 1 {
			bottomLine += theme.StyleDim.Render("s")
		}
		bottomLine += theme.StyleDim.Render("  y=yank  esc=cancel")
	case r.leaping && r.leapChar == "":
		bottomLine = lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render("s›") +
			lipgloss.NewStyle().Foreground(theme.Overlay1).Render("█  type to seek, esc to cancel")
	case r.leaping:
		bottomLine = lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render("s›"+r.leapChar) + "█" +
			lipgloss.NewStyle().Foreground(theme.Overlay1).Render(fmt.Sprintf("  %d targets — type more or a label, enter=first, esc=cancel", len(r.leapTargets)))
	case r.searching:
		prefix := "/"
		if !r.searchForward {
			prefix = "?"
		}
		bottomLine = lipgloss.NewStyle().Foreground(theme.Text).Render(prefix+r.searchBuf) + "█"
	default:
		matchInfo := ""
		if r.searchQuery != "" && len(r.matches) > 0 {
			matchInfo = fmt.Sprintf(" [%d/%d]", r.matchIdx+1, len(r.matches))
		} else if r.searchQuery != "" {
			matchInfo = " [no matches]"
		}
		hint := theme.StyleDim.Render(fmt.Sprintf("── %d%% (%d/%d lines)%s", pct, r.cursor+1, len(r.lines), matchInfo))
		if focused {
			hint = lipgloss.NewStyle().Foreground(theme.Overlay2).Render(
				fmt.Sprintf("── %d%% (%d/%d lines)%s  j/k ½d/u pg f/b  s leap  / search  G bot", pct, r.cursor+1, len(r.lines), matchInfo),
			)
		}
		bottomLine = hint
	}

	return strings.Join(append([]string{header}, append(rows, bottomLine)...), "\n")
}

// renderLeapLine builds a line string where the matched positions show labels
// and all other characters are dimmed.
func renderLeapLine(plain string, targets []leapTarget, dimStyle, labelStyle lipgloss.Style, maxW int) string {
	// Build a map col→label for this line.
	labelAt := map[int]string{}
	for _, t := range targets {
		labelAt[t.col] = t.label
	}

	runes := []rune(plain)
	var b strings.Builder
	i := 0
	for i < len(runes) {
		byteOff := len(string(runes[:i]))
		if lbl, ok := labelAt[byteOff]; ok {
			b.WriteString(labelStyle.Render(lbl))
			i++ // skip the matched character, replaced by label
		} else {
			b.WriteString(dimStyle.Render(string(runes[i])))
			i++
		}
	}
	return truncate(b.String(), maxW)
}

func isMatch(lines []string, idx int, query string) bool {
	if idx < 0 || idx >= len(lines) {
		return false
	}
	return strings.Contains(strings.ToLower(stripStyle(lines[idx])), strings.ToLower(query))
}

// highlightMatch wraps the first occurrence of query in the line with style.
func highlightMatch(line, query string, style lipgloss.Style, maxW int) string {
	plain := stripStyle(line)
	lower := strings.ToLower(plain)
	q := strings.ToLower(query)
	idx := strings.Index(lower, q)
	if idx < 0 {
		return truncate(line, maxW)
	}
	before := plain[:idx]
	match := plain[idx : idx+len(q)]
	after := plain[idx+len(q):]
	return truncate(before+style.Render(match)+after, maxW)
}

// buildLines renders the response into wrapped lines without height clamping.
func buildLines(resp *httpx.Response, resolved *httpx.ResolvedRequest, width int) []string {
	var b strings.Builder

	if resp == nil {
		return nil
	}

	if resp.Err != nil {
		b.WriteString(theme.StyleError.Render("ERR ") + resp.Err.Error() + "\n")
	} else {
		st := theme.StyleSuccess
		if resp.StatusCode >= 500 {
			st = theme.StyleError
		} else if resp.StatusCode >= 400 {
			st = theme.StyleWarning
		}
		b.WriteString(st.Bold(true).Render(fmt.Sprintf("%s ", resp.Status)))
		b.WriteString(theme.StyleDim.Render(fmt.Sprintf("(%s)", resp.Elapsed.Round(1e6))))
		b.WriteString("\n")
	}
	b.WriteString(theme.MethodStyle(resp.Method).Bold(true).Render(resp.Method) + theme.StyleDim.Render(" "+resp.URL) + "\n")

	// Show the sent request body so the user can verify variable resolution.
	if resolved != nil && len(resolved.Body) > 0 {
		b.WriteString(theme.StyleSubtext.Bold(true).Render("Sent") + "\n")
		b.WriteString(pretty(resolved.Body, resolved.BodyType) + "\n")
	}
	b.WriteString("\n")

	if len(resp.Body) > 0 {
		b.WriteString(pretty(resp.Body, resp.Headers.Get("Content-Type")))
	}

	raw := wrapLines(b.String(), width)
	return strings.Split(raw, "\n")
}

func pretty(body []byte, contentType string) string {
	// Determine lexer by content-type then by content sniffing.
	var lexerName string
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "json") || looksJSON(body):
		lexerName = "json"
		// Pretty-print JSON before highlighting.
		var v any
		if err := json.Unmarshal(body, &v); err == nil {
			if out, err := json.MarshalIndent(v, "", "  "); err == nil {
				body = out
			}
		}
	case strings.Contains(ct, "xml") || strings.Contains(ct, "html"):
		lexerName = "xml"
	case strings.Contains(ct, "yaml"):
		lexerName = "yaml"
	}

	if lexerName != "" {
		if highlighted := chromaHighlight(body, lexerName); highlighted != "" {
			return highlighted
		}
	}
	return string(body)
}

// chromaHighlight renders body with Chroma using the terminal256 formatter and
// the Catppuccin Macchiato style. Returns "" on any error.
func chromaHighlight(body []byte, lexerName string) string {
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get("catppuccin-macchiato")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	iterator, err := lexer.Tokenise(nil, string(body))
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return ""
	}
	// Chroma appends a trailing newline; trim it so callers decide.
	return strings.TrimRight(buf.String(), "\n")
}

func looksJSON(b []byte) bool {
	s := strings.TrimSpace(string(b))
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

// buildScrollbar returns a slice of single-character strings (one per viewport row)
// representing a proportional scrollbar.
func buildScrollbar(offset, totalLines, viewH int) []string {
	bar := make([]string, viewH)
	track := theme.StyleDim.Render("│")
	thumb := lipgloss.NewStyle().Foreground(theme.Overlay2).Render("┃")

	if totalLines <= viewH {
		for i := range bar {
			bar[i] = track
		}
		return bar
	}

	thumbH := viewH * viewH / totalLines
	if thumbH < 1 {
		thumbH = 1
	}
	maxOffset := totalLines - viewH
	thumbTop := offset * (viewH - thumbH) / maxOffset

	for i := range bar {
		if i >= thumbTop && i < thumbTop+thumbH {
			bar[i] = thumb
		} else {
			bar[i] = track
		}
	}
	return bar
}
