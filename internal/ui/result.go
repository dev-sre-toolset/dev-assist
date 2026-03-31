package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dev-sre-toolset/dev-assist/internal/tools"
)

// ResultModel wraps a scrollable viewport for tool output.
type ResultModel struct {
	tool     *tools.Tool
	viewport viewport.Model
	content  string
	errMsg   string
	width    int
	height   int
}

func NewResultModel(t *tools.Tool, content string, errMsg string, width, height int) ResultModel {
	vp := viewport.New(width-4, height-8)
	vp.Style = resultBorderStyle

	m := ResultModel{
		tool:    t,
		content: content,
		errMsg:  errMsg,
		width:   width,
		height:  height,
	}

	if errMsg != "" {
		vp.SetContent(errorStyle.Render("✗ Error\n\n") + errorStyle.Render(errMsg))
	} else {
		vp.SetContent(content)
	}
	m.viewport = vp
	return m
}

func (m ResultModel) Update(msg tea.Msg) (ResultModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *ResultModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w - 4
	m.viewport.Height = h - 8
}

func (m ResultModel) View() string {
	var sb strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	sb.WriteString("\n  ")
	if m.errMsg != "" {
		sb.WriteString(neonBadge(" ✗ ERROR ", colorOrange))
		sb.WriteString("  " + errorStyle.Render(m.tool.Name))
	} else {
		sb.WriteString(neonBadge(" ✓ RESULT ", colorGreen))
		sb.WriteString("  " + resultHeaderStyle.Render(m.tool.Name))
	}
	sb.WriteString("\n\n")

	// Scrollable content
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Scroll indicator
	pct := m.viewport.ScrollPercent()
	indicator := fmt.Sprintf("  %.0f%%", pct*100)
	if m.viewport.AtBottom() {
		indicator += " (end)"
	}
	sb.WriteString(subtitleStyle.Render(indicator))
	sb.WriteString("\n")

	// Help bar
	help := strings.Join([]string{
		helpItem("↑↓/k/j", "scroll"),
		helpItem("g/G", "top/bottom"),
		helpItem("esc", "back to input"),
		helpItem("m", "main menu"),
		helpItem("q", "quit"),
	}, "  ")
	sb.WriteString("\n" + helpDescStyle.Render("  ") + help + "\n")

	return sb.String()
}
