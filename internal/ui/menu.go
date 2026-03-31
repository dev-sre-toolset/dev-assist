package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dev-sre-toolset/dev-assist/internal/tools"
)

// menuItem wraps a *tools.Tool to satisfy bubbles/list.Item.
type menuItem struct {
	tool *tools.Tool
}

func (m menuItem) Title() string       { return m.tool.Name }
func (m menuItem) Description() string { return m.tool.Description }
func (m menuItem) FilterValue() string { return m.tool.Name + " " + m.tool.Category + " " + m.tool.Description }

// ── custom delegate ───────────────────────────────────────────────────────────

type menuDelegate struct{}

func (d menuDelegate) Height() int                             { return 2 }
func (d menuDelegate) Spacing() int                            { return 0 }
func (d menuDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d menuDelegate) Render(w fmt.Stringer, m list.Model, index int, item list.Item) {
	// Render is a no-op — we draw the full menu ourselves for category grouping.
}

// ── MenuModel ─────────────────────────────────────────────────────────────────

// MenuModel is the interactive tool-selection list.
type MenuModel struct {
	tools    []*tools.Tool
	cursor   int
	width    int
	height   int
	filter   string
	filtered []*tools.Tool
}

func NewMenuModel() MenuModel {
	m := MenuModel{
		tools:  tools.Registry,
		cursor: 0,
	}
	m.filtered = m.tools
	return m
}

func (m MenuModel) Selected() *tools.Tool {
	if len(m.filtered) == 0 {
		return nil
	}
	return m.filtered[m.cursor]
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
		default:
			// Printable single character — add to filter
			if len(msg.Runes) == 1 {
				m.filter += string(msg.Runes)
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func (m *MenuModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.tools
	} else {
		lower := strings.ToLower(m.filter)
		var out []*tools.Tool
		for _, t := range m.tools {
			if strings.Contains(strings.ToLower(t.Name), lower) ||
				strings.Contains(strings.ToLower(t.Category), lower) ||
				strings.Contains(strings.ToLower(t.Description), lower) {
				out = append(out, t)
			}
		}
		m.filtered = out
	}
	m.cursor = 0
}

func (m *MenuModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m MenuModel) View() string {
	var sb strings.Builder

	// ── Logo + tagline ────────────────────────────────────────────────────────
	sb.WriteString(logoStyle.Render(logo))
	sb.WriteString("\n")
	sb.WriteString(logoSubStyle.Render("SRE Utility Belt  •  decode · inspect · analyse · assist"))
	sb.WriteString("\n\n")

	// ── Filter bar ────────────────────────────────────────────────────────────
	if m.filter != "" {
		filterBadge := neonBadge("filter", colorMagenta)
		sb.WriteString("  " + filterBadge + " " + lipgloss.NewStyle().
			Foreground(colorYellow).Bold(true).Render(m.filter+"_") + "\n\n")
	} else {
		sb.WriteString(inputHintStyle.Render("  › type to filter tools") + "\n\n")
	}

	// ── Tool list grouped by category ─────────────────────────────────────────
	// Map categories to neon badge colours for visual variety
	catColour := map[string]lipgloss.Color{
		"SSL & Certificates": colorCyan,
		"Auth & Tokens":      colorMagenta,
		"Network":            colorGreen,
		"Data":               colorYellow,
	}

	var lastCat string
	visibleItems := 0
	maxVisible := m.height - 14 // leave room for logo, filter bar, help bar

	for i, t := range m.filtered {
		if visibleItems >= maxVisible {
			remaining := len(m.filtered) - i
			sb.WriteString(menuDescStyle.Render(fmt.Sprintf("  … %d more (filter to narrow)\n", remaining)))
			break
		}

		if t.Category != lastCat {
			colour, ok := catColour[t.Category]
			if !ok {
				colour = colorPurple
			}
			badge := neonBadge(" "+t.Category+" ", colour)
			sb.WriteString("\n  " + badge + "\n")
			lastCat = t.Category
		}

		if i == m.cursor {
			// Selected row: neon cyan left bar + highlighted name + dimmed desc
			name := menuItemSelectedStyle.Render(" ▶  " + t.Name)
			desc := menuDescStyle.Render("      " + t.Description)
			sb.WriteString(name + "\n")
			sb.WriteString(desc + "\n")
		} else {
			sb.WriteString(menuItemStyle.Render("    " + t.Name) + "\n")
		}
		visibleItems++
	}

	if len(m.filtered) == 0 {
		sb.WriteString("\n  " + errorStyle.Render("✗  no tools match — press backspace to clear filter") + "\n")
	}

	// ── Help bar ──────────────────────────────────────────────────────────────
	sb.WriteString("\n")
	help := strings.Join([]string{
		helpItem("↑↓", "navigate"),
		helpItem("enter", "select"),
		helpItem("type", "filter"),
		helpItem("bksp", "clear"),
		helpItem("q", "quit"),
	}, "   ")
	sb.WriteString("  " + help + "\n")

	return sb.String()
}
