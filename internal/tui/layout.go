package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/theme"
)

// renderLayout assembles the final view from the model state.
func (m *Model) renderLayout() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}

	statusBar := m.renderStatusBar()
	cmdLine := m.renderCommandLine()

	// Reserve 1 line for status bar + 1 for command/status line.
	bodyHeight := m.height - 2
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	// Sidebar: ~30 cols (or 25% of width).
	sidebarW := 30
	if m.width/4 > sidebarW {
		sidebarW = m.width / 4
	}
	if sidebarW > m.width-30 {
		sidebarW = m.width / 3
	}
	rightW := m.width - sidebarW - 1
	if rightW < 20 {
		rightW = 20
	}

	// Left sidebar: tree on top, env pane at the bottom.
	envH := 8
	if bodyHeight/4 > envH {
		envH = bodyHeight / 4
	}
	treeH := bodyHeight - envH

	// Right side: top half = Request, bottom half = Response (full width).
	reqHeight := bodyHeight / 2
	respHeight := bodyHeight - reqHeight

	// Cache geometry for mouse hit-testing.
	m.geom = paneGeometry{
		sidebarW:  sidebarW,
		treeH:     treeH,
		envH:      envH,
		reqHeight: reqHeight,
	}

	treeView := m.tree.View(sidebarW-2, treeH-2, m.focused == PaneTree)
	envView := m.env.View(m.activeEnvName(), sidebarW-4, m.focused == PaneEnv)
	sidebar := lipgloss.JoinVertical(lipgloss.Left,
		boxed(treeView, sidebarW, treeH, m.focused == PaneTree),
		boxed(envView, sidebarW, envH, m.focused == PaneEnv),
	)

	var reqView, respView string
	req, scope := m.activeRequestAndScope()
	reqView = m.request.View(req, scope, m.activeEnvName(), rightW-4, reqHeight-2, m.focused == PaneRequest)
	respView = m.response.View(rightW-4, respHeight-2, m.focused == PaneResponse)

	right := lipgloss.JoinVertical(lipgloss.Left,
		boxed(reqView, rightW, reqHeight, m.focused == PaneRequest),
		boxed(respView, rightW, respHeight, m.focused == PaneResponse),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)

	// Modal overlays.
	if m.history.Visible {
		modal := m.history.View(m.width-8, m.height-8)
		body = overlay(body, modal, m.width, m.height)
	}
	if m.vars.Visible {
		v := m.vars.View(50)
		body = overlay(body, v, m.width, m.height)
	}
	if m.help.Visible {
		h := m.help.View(m.width/3, m.height-4)
		body = overlayBottomRight(body, h, m.width, m.height)
	}

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, body, cmdLine)
}

func boxed(content string, w, h int, focused bool) string {
	color := theme.BorderUnfocused
	if focused {
		color = theme.BorderFocused
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Width(w - 2).
		Height(h - 2)
	return style.Render(content)
}

// overlay places ov centered over body.
func overlay(body, ov string, w, h int) string {
	bodyLines := strings.Split(body, "\n")
	ovLines := strings.Split(ov, "\n")
	startRow := (h - len(ovLines)) / 2
	if startRow < 0 {
		startRow = 0
	}
	for i, ovl := range ovLines {
		row := startRow + i
		if row < 0 || row >= len(bodyLines) {
			continue
		}
		ovWidth := lipgloss.Width(ovl)
		col := (w - ovWidth) / 2
		if col < 0 {
			col = 0
		}
		bodyLines[row] = mergeLine(bodyLines[row], ovl, col, w)
	}
	return strings.Join(bodyLines, "\n")
}

// overlayBottomRight places ov in the bottom-right corner of body,
// leaving a 1-line gap above the command line (h already excludes it).
func overlayBottomRight(body, ov string, w, h int) string {
	bodyLines := strings.Split(body, "\n")
	ovLines := strings.Split(ov, "\n")
	// Pin the bottom of the modal to the last body row (command line is outside body).
	startRow := len(bodyLines) - len(ovLines)
	if startRow < 0 {
		startRow = 0
	}
	for i, ovl := range ovLines {
		row := startRow + i
		if row < 0 || row >= len(bodyLines) {
			continue
		}
		ovWidth := lipgloss.Width(ovl)
		col := w - ovWidth
		if col < 0 {
			col = 0
		}
		bodyLines[row] = mergeLine(bodyLines[row], ovl, col, w)
	}
	return strings.Join(bodyLines, "\n")
}

// mergeLine writes ov into base starting at column col (display width based,
// best-effort: it pads base with spaces to col, then concatenates ov, then
// fills remaining width). Existing styling on base is mostly lost in the
// overlapped region — acceptable for modal overlays.
func mergeLine(base, ov string, col, totalW int) string {
	plainBase := stripANSI(base)
	runes := []rune(plainBase)
	if col > len(runes) {
		// Pad.
		runes = append(runes, []rune(strings.Repeat(" ", col-len(runes)))...)
	}
	prefix := string(runes[:col])
	rest := ""
	tailStart := col + lipgloss.Width(ov)
	if tailStart < len(runes) {
		rest = string(runes[tailStart:])
	}
	out := prefix + ov + rest
	// Truncate if too wide.
	if lipgloss.Width(out) > totalW {
		// Naive: trim runes off the end.
		ro := []rune(out)
		for lipgloss.Width(string(ro)) > totalW && len(ro) > 0 {
			ro = ro[:len(ro)-1]
		}
		out = string(ro)
	}
	return out
}

func stripANSI(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == 0x1b {
			in = true
			continue
		}
		if in {
			if r == 'm' {
				in = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
