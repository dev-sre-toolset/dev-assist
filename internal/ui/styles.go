package ui

import "github.com/charmbracelet/lipgloss"

// ── Neon colour palette ───────────────────────────────────────────────────────
// Background tones
var (
	colorBg      = lipgloss.Color("#0d0e1a") // near-black with a blue tint
	colorSurface = lipgloss.Color("#12131f") // slightly lighter panel
	colorBorder  = lipgloss.Color("#2a2b3d") // dim border

	// Neon accents
	colorCyan    = lipgloss.Color("#00f5ff") // electric cyan  — primary accent
	colorMagenta = lipgloss.Color("#ff2d95") // hot pink       — secondary accent
	colorGreen   = lipgloss.Color("#39ff14") // neon green     — success / result
	colorYellow  = lipgloss.Color("#ffe600") // electric yellow — warning / highlight
	colorPurple  = lipgloss.Color("#bf5fff") // neon purple    — categories
	colorOrange  = lipgloss.Color("#ff6b35") // neon orange    — errors

	// Text
	colorFg    = lipgloss.Color("#e2e8f0") // primary text (near-white)
	colorFgDim = lipgloss.Color("#8892a4") // secondary text
	colorMuted = lipgloss.Color("#4a5568") // placeholder / hints
)

// ── App chrome ────────────────────────────────────────────────────────────────

var titleStyle = lipgloss.NewStyle().
	Foreground(colorCyan).
	Bold(true).
	Padding(0, 1)

var subtitleStyle = lipgloss.NewStyle().
	Foreground(colorMuted)

var appBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorCyan)

// ── Menu ─────────────────────────────────────────────────────────────────────

var menuItemStyle = lipgloss.NewStyle().
	Foreground(colorFgDim).
	PaddingLeft(2)

var menuItemSelectedStyle = lipgloss.NewStyle().
	Foreground(colorCyan).
	Bold(true).
	PaddingLeft(1).
	Border(lipgloss.Border{Left: "▌"}, false, false, false, true).
	BorderForeground(colorCyan)

var categoryStyle = lipgloss.NewStyle().
	Foreground(colorPurple).
	Bold(true).
	PaddingLeft(2).
	MarginTop(1)

var menuDescStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	PaddingLeft(4)

// ── Input ─────────────────────────────────────────────────────────────────────

var inputLabelStyle = lipgloss.NewStyle().
	Foreground(colorCyan).
	Bold(true)

var inputHintStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Italic(true)

var activeInputBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorCyan)

var inactiveInputBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder)

var toggleActiveStyle = lipgloss.NewStyle().
	Foreground(colorBg).
	Background(colorCyan).
	Bold(true).
	Padding(0, 1)

var toggleInactiveStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Background(colorSurface).
	Padding(0, 1)

var errorStyle = lipgloss.NewStyle().
	Foreground(colorOrange).
	Bold(true)

// ── Result ───────────────────────────────────────────────────────────────────

var resultHeaderStyle = lipgloss.NewStyle().
	Foreground(colorGreen).
	Bold(true)

var resultBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorGreen)

// ── Help bar ─────────────────────────────────────────────────────────────────

var helpKeyStyle = lipgloss.NewStyle().
	Foreground(colorYellow).
	Bold(true)

var helpDescStyle = lipgloss.NewStyle().
	Foreground(colorMuted)

// ── Banner / logo ─────────────────────────────────────────────────────────────

const logo = `
  █████╗ ███████╗███████╗██╗███████╗████████╗    ███╗   ███╗███████╗
 ██╔══██╗██╔════╝██╔════╝██║██╔════╝╚══██╔══╝   ████╗ ████║██╔════╝
 ███████║███████╗███████╗██║███████╗   ██║  █████╗██╔████╔██║█████╗
 ██╔══██║╚════██║╚════██║██║╚════██║   ██║  ╚════╝██║╚██╔╝██║██╔══╝
 ██║  ██║███████║███████║██║███████║   ██║        ██║ ╚═╝ ██║███████╗
 ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝╚══════╝   ╚═╝        ╚═╝     ╚═╝╚══════╝`

var logoStyle = lipgloss.NewStyle().Foreground(colorCyan)
var logoSubStyle = lipgloss.NewStyle().
	Foreground(colorMagenta).
	Bold(true).
	Padding(0, 2)

// ── Utilities ────────────────────────────────────────────────────────────────

func helpItem(key, desc string) string {
	return helpKeyStyle.Render(key) + " " + helpDescStyle.Render(desc)
}

// neonBadge renders a small pill badge with a neon colour.
func neonBadge(text string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(colorBg).
		Background(color).
		Bold(true).
		Padding(0, 1).
		Render(text)
}

// dimSeparator renders a full-width horizontal rule.
func dimSeparator(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Foreground(colorBorder).Render(
		lipgloss.NewStyle().Width(width).Render(""),
	)
}
