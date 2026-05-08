package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/interp"
	"github.com/luca-trifilio/brio/internal/theme"
)

// RequestModel holds rendered lines and vim scroll state for the Request pane.
type RequestModel struct {
	req     *canonical.Request
	scope   *interp.VarScope
	lastEnv string // last env name used for line-building (triggers rebuild on change)

	lines  []string
	offset int
	cursor int
	height int
	width  int
	count  int
}

func NewRequest() *RequestModel { return &RequestModel{} }

// viewH returns the number of scrollable content rows.
// Layout: 2 header lines (title + separator) + viewH content rows + 1 bottom-hint row = height.
func (r *RequestModel) viewH() int {
	h := r.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (r *RequestModel) clamp() {
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
	if len(r.lines) > 0 && r.cursor >= len(r.lines) {
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

func (r *RequestModel) n() int {
	if r.count <= 0 {
		return 1
	}
	return r.count
}

func (r *RequestModel) halfPage() int {
	h := r.viewH() / 2
	if h < 1 {
		h = 1
	}
	return h
}

func (r *RequestModel) rebuild() {
	// Wrap at contentW (width minus 1 scrollbar column) so pad() in View
	// never needs to truncate a just-wrapped line.
	wrapW := r.width - 1
	if wrapW < 1 {
		wrapW = 1
	}
	r.lines = buildRequestLines(r.req, r.scope, wrapW)
}

// HandleKey processes a vim motion key. Returns true if the key was consumed.
func (r *RequestModel) HandleKey(key string) bool {
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
	default:
		return false
	}
	return true
}

// View renders the request pane with scrolling, scrollbar, and syntax-highlighted body.
// req, scope, and envName are passed on every frame so the model can detect changes
// and rebuild its line buffer without the caller managing lifecycle.
func (r *RequestModel) View(req *canonical.Request, scope *interp.VarScope, envName string, width, height int, focused bool) string {
	reqChanged := req != r.req
	envChanged := envName != r.lastEnv
	dimChanged := width != r.width || height != r.height

	switch {
	case reqChanged:
		// New request selected — rebuild and reset scroll to top.
		r.req = req
		r.scope = scope
		r.lastEnv = envName
		r.width = width
		r.height = height
		r.offset = 0
		r.cursor = 0
		r.count = 0
		r.rebuild()
	case envChanged || dimChanged:
		// Same request but env or size changed — rebuild, keep scroll position.
		r.scope = scope
		r.lastEnv = envName
		r.width = width
		r.height = height
		r.rebuild()
		r.clamp()
	}

	title := "Request"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}
	separator := theme.StyleDim.Render(strings.Repeat("─", width))
	header := theme.StyleTitle.Render(title) + "\n" + separator

	viewH := r.viewH()

	if len(r.lines) == 0 {
		return header + "\n" + theme.StyleDim.Render("  (no request selected)")
	}

	end := r.offset + viewH
	if end > len(r.lines) {
		end = len(r.lines)
	}
	visible := make([]string, viewH)
	copy(visible, r.lines[r.offset:end])

	scrollbar := buildScrollbar(r.offset, len(r.lines), viewH)

	scrollable := len(r.lines) > viewH
	pct := 100
	if scrollable {
		pct = r.offset * 100 / (len(r.lines) - viewH)
		if pct > 100 {
			pct = 100
		}
	}

	contentW := width - 1 // 1 col for scrollbar
	if contentW < 1 {
		contentW = 1
	}

	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Sky)

	rows := make([]string, viewH)
	for i, line := range visible {
		bar := " "
		if scrollable && i < len(scrollbar) {
			bar = scrollbar[i]
		}
		absLine := r.offset + i
		cell := pad(line, contentW)
		if focused && absLine == r.cursor {
			cell = cursorStyle.Render(cell)
		}
		rows[i] = cell + bar
	}

	var bottomLine string
	if focused {
		hint := fmt.Sprintf("── %d%% (%d/%d lines)  j/k  d/u  f/b  G bot", pct, r.cursor+1, len(r.lines))
		bottomLine = lipgloss.NewStyle().Foreground(theme.Overlay2).Render(hint)
	} else {
		bottomLine = theme.StyleDim.Render(fmt.Sprintf("── %d%% (%d/%d lines)", pct, r.cursor+1, len(r.lines)))
	}

	return strings.Join(append([]string{header}, append(rows, bottomLine)...), "\n")
}

// buildRequestLines renders the request into syntax-highlighted, width-wrapped lines.
func buildRequestLines(req *canonical.Request, scope *interp.VarScope, width int) []string {
	if req == nil {
		return []string{theme.StyleDim.Render("  (no request selected)")}
	}

	interpl := func(s string) string {
		if scope == nil {
			return s
		}
		return scope.Interpolate(s)
	}

	var b strings.Builder

	method := string(req.Method)
	url := interpl(req.URL)
	b.WriteString(theme.MethodStyle(method).Bold(true).Render(method) + " " + theme.StyleText.Render(url) + "\n")
	b.WriteString(theme.StyleDim.Render(req.SourcePath) + "\n")
	b.WriteString("\n")

	if len(req.Headers) > 0 {
		b.WriteString(theme.StyleSubtext.Bold(true).Render("Headers") + "\n")
		for _, h := range req.Headers {
			if h.Disabled {
				continue
			}
			b.WriteString("  " + theme.StyleFocused.Render(h.Name) + ": " + interpl(h.Value) + "\n")
		}
		b.WriteString("\n")
	}

	if len(req.QueryParams) > 0 {
		b.WriteString(theme.StyleSubtext.Bold(true).Render("Query") + "\n")
		for _, p := range req.QueryParams {
			if p.Disabled {
				continue
			}
			b.WriteString("  " + theme.StyleFocused.Render(p.Name) + " = " + interpl(p.Value) + "\n")
		}
		b.WriteString("\n")
	}

	if req.Body.Type != "" && req.Body.Raw != "" {
		b.WriteString(theme.StyleSubtext.Bold(true).Render("Body ("+req.Body.Type+")") + "\n")
		raw := interpl(req.Body.Raw)
		highlighted := highlightRequestBody(raw, req.Body.Type)
		b.WriteString(indent(highlighted, "  "))
		b.WriteString("\n")
	}

	if req.Scripts.Pre != "" {
		b.WriteString(theme.StyleWarning.Render("(has pre-request script — not executed in MVP)") + "\n")
	}

	raw := wrapLines(b.String(), width)
	return strings.Split(raw, "\n")
}

// highlightRequestBody applies Chroma syntax highlighting to a request body.
// Bruno body types ("json", "xml", "graphql") are mapped to content-type hints
// consumed by pretty() which also pretty-prints JSON.
func highlightRequestBody(raw, bodyType string) string {
	switch bodyType {
	case "json":
		return pretty([]byte(raw), "application/json")
	case "xml":
		return pretty([]byte(raw), "application/xml")
	case "graphql":
		return pretty([]byte(raw), "application/graphql")
	default:
		return raw
	}
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func wrapLines(s string, width int) string {
	if width <= 0 {
		return s
	}
	// Hardwrap preserves ANSI codes and leading whitespace (indented JSON, headers…).
	// Each original newline is kept; long lines are split into additional lines.
	return ansi.Hardwrap(s, width, true)
}
