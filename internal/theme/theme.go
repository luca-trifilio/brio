// Package theme defines the Catppuccin Macchiato color palette and shared
// lipgloss styles for brio.
package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Macchiato palette (hex values from https://catppuccin.com/palette)
const (
	Rosewater = lipgloss.Color("#f4dbd6")
	Flamingo  = lipgloss.Color("#f0c6c6")
	Pink      = lipgloss.Color("#f5bde6")
	Mauve     = lipgloss.Color("#c6a0f6")
	Red       = lipgloss.Color("#ed8796")
	Maroon    = lipgloss.Color("#ee99a0")
	Peach     = lipgloss.Color("#f5a97f")
	Yellow    = lipgloss.Color("#eed49f")
	Green     = lipgloss.Color("#a6da95")
	Teal      = lipgloss.Color("#8bd5ca")
	Sky       = lipgloss.Color("#91d7e3")
	Sapphire  = lipgloss.Color("#7dc4e4")
	Blue      = lipgloss.Color("#8aadf4")
	Lavender  = lipgloss.Color("#b7bdf8")

	Text      = lipgloss.Color("#cad3f5")
	Subtext1  = lipgloss.Color("#b8c0e0")
	Subtext0  = lipgloss.Color("#a5adcb")
	Overlay2  = lipgloss.Color("#939ab7")
	Overlay1  = lipgloss.Color("#8087a2")
	Overlay0  = lipgloss.Color("#6e738d")
	Surface2  = lipgloss.Color("#5b6078")
	Surface1  = lipgloss.Color("#494d64")
	Surface0  = lipgloss.Color("#363a4f")
	Base      = lipgloss.Color("#24273a")
	Mantle    = lipgloss.Color("#1e2030")
	Crust     = lipgloss.Color("#181926")
)

// Semantic aliases — use these in UI code instead of raw palette names.
var (
	// Text hierarchy
	StyleText     = lipgloss.NewStyle().Foreground(Text)
	StyleSubtext  = lipgloss.NewStyle().Foreground(Subtext0)
	StyleDim      = lipgloss.NewStyle().Foreground(Overlay1)
	StyleBold     = lipgloss.NewStyle().Foreground(Text).Bold(true)

	// Accents
	StyleTitle      = lipgloss.NewStyle().Foreground(Lavender).Bold(true)
	StyleCollection = lipgloss.NewStyle().Foreground(Peach).Bold(true)  // collection root names
	StyleFocused    = lipgloss.NewStyle().Foreground(Blue)
	StyleActive     = lipgloss.NewStyle().Foreground(Green).Bold(true)
	StyleCursor     = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	StyleSuccess    = lipgloss.NewStyle().Foreground(Green)
	StyleError      = lipgloss.NewStyle().Foreground(Red)
	StyleWarning    = lipgloss.NewStyle().Foreground(Yellow)
	StyleHint       = lipgloss.NewStyle().Foreground(Overlay1)

	// Cursor line — used uniformly across all panes for selected rows.
	StyleCursorLine = lipgloss.NewStyle().
			Background(Surface1).
			Foreground(Sky).
			Bold(true)

	// HTTP methods
	StyleGET    = lipgloss.NewStyle().Foreground(Green)
	StylePOST   = lipgloss.NewStyle().Foreground(Blue)
	StylePUT    = lipgloss.NewStyle().Foreground(Yellow)
	StylePATCH  = lipgloss.NewStyle().Foreground(Peach)
	StyleDELETE = lipgloss.NewStyle().Foreground(Red)

	// Status bar
	StyleStatusBar = lipgloss.NewStyle().
			Background(Mantle).
			Foreground(Text).
			Padding(0, 1)
	StyleModeNormal  = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	StyleModeInsert  = lipgloss.NewStyle().Foreground(Green).Bold(true)
	StyleModeCommand = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	StyleModeVisual  = lipgloss.NewStyle().Foreground(Mauve).Bold(true)

	// Visual selection line — distinct from cursor (Surface1/Sky) and search (Surface0/Yellow).
	StyleVisualLine = lipgloss.NewStyle().Background(Surface0).Foreground(Lavender)
	// Cursor line within a visual selection — anchor end indicator.
	StyleVisualCursor = lipgloss.NewStyle().Background(Surface1).Foreground(Mauve).Bold(true)

	// Borders
	BorderFocused  = Blue
	BorderUnfocused = Surface2
)

// ----------------------------------------------------------------------------
// Environment safety tiers
// ----------------------------------------------------------------------------

// EnvTier classifies an environment by its risk level.
type EnvTier int

const (
	TierSafe    EnvTier = iota // local, test, dev, …
	TierCaution                // staging, uat, pre-prod, …
	TierDanger                 // prod, production, live, …
)

// ClassifyEnv maps an environment name to its safety tier.
// Matching is case-insensitive substring search.
func ClassifyEnv(name string) EnvTier {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "prod") ||
		strings.Contains(lower, "live"):
		return TierDanger
	case strings.Contains(lower, "stag") ||
		strings.Contains(lower, "uat") ||
		strings.Contains(lower, "pre"):
		return TierCaution
	default:
		return TierSafe
	}
}

// EnvTierIcon returns a styled single-character indicator for the env tier.
//
//	● green  → safe  (local / test / dev)
//	▲ orange → caution (staging / uat / pre-prod)
//	⚠ red   → danger  (prod / live)
func EnvTierIcon(name string) string {
	switch ClassifyEnv(name) {
	case TierDanger:
		return lipgloss.NewStyle().Foreground(Red).Bold(true).Render("⚠")
	case TierCaution:
		return lipgloss.NewStyle().Foreground(Peach).Render("▲")
	default:
		return lipgloss.NewStyle().Foreground(Green).Render("●")
	}
}

// EnvTierStyle returns the lipgloss text style for an environment name.
func EnvTierStyle(name string) lipgloss.Style {
	switch ClassifyEnv(name) {
	case TierDanger:
		return lipgloss.NewStyle().Foreground(Red).Bold(true)
	case TierCaution:
		return lipgloss.NewStyle().Foreground(Peach)
	default:
		return lipgloss.NewStyle().Foreground(Green)
	}
}

// EnvBadge renders "icon name" styled for the env tier,
// suitable for inline use in the status bar.
func EnvBadge(name string) string {
	if name == "" {
		return StyleDim.Render("(no env)")
	}
	return EnvTierIcon(name) + " " + EnvTierStyle(name).Render(name)
}

// IsMutatingMethod reports whether method is one of POST, PUT or PATCH.
func IsMutatingMethod(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH":
		return true
	}
	return false
}

// MutatingMethodsBlocked reports whether POST/PUT/PATCH should be
// blocked for the given environment (i.e. it is a TierDanger env).
func MutatingMethodsBlocked(envName string) bool {
	return ClassifyEnv(envName) == TierDanger
}

// ----------------------------------------------------------------------------
// HTTP method styles
// ----------------------------------------------------------------------------

// MethodStyle returns the style for a given HTTP method string.
func MethodStyle(method string) lipgloss.Style {
	switch method {
	case "GET":
		return StyleGET
	case "POST":
		return StylePOST
	case "PUT":
		return StylePUT
	case "PATCH":
		return StylePATCH
	case "DELETE":
		return StyleDELETE
	default:
		return StyleSubtext
	}
}
