// Package ui implements the BubbleTea TUI for dev-assist.
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dev-sre-toolset/dev-assist/internal/tools"
)

type appState int

const (
	stateMenu   appState = iota
	stateInput
	stateResult
)

// App is the root BubbleTea model.
type App struct {
	state  appState
	width  int
	height int
	menu   MenuModel
	input  InputModel
	result ResultModel
}

// NewApp constructs the root model. Call with a specific tool ID to open that
// tool directly; pass "" to start at the menu.
func NewApp() App {
	return App{
		state: stateMenu,
		menu:  NewMenuModel(),
	}
}

// NewAppForTool opens the TUI directly on the input screen for a specific tool.
func NewAppForTool(t *tools.Tool) App {
	a := NewApp()
	a.state = stateInput
	a.input = NewInputModel(t, 80, 24)
	return a
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.menu.SetSize(msg.Width, msg.Height)
		a.input.SetSize(msg.Width, msg.Height)
		a.result.SetSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		switch a.state {
		case stateMenu:
			switch msg.String() {
			case "q":
				return a, tea.Quit
			case "enter":
				t := a.menu.Selected()
				if t != nil {
					a.input = NewInputModel(t, a.width, a.height)
					a.state = stateInput
				}
			default:
				var cmd tea.Cmd
				a.menu, cmd = a.menu.Update(msg)
				return a, cmd
			}

		case stateInput:
			switch msg.String() {
			case "esc":
				a.state = stateMenu
				return a, nil
			default:
				var cmd tea.Cmd
				a.input, cmd = a.input.Update(msg)
				return a, cmd
			}

		case stateResult:
			switch msg.String() {
			case "esc":
				a.state = stateInput
				return a, nil
			case "m":
				a.state = stateMenu
				return a, nil
			case "q":
				return a, tea.Quit
			case "g":
				a.result.viewport.GotoTop()
			case "G":
				a.result.viewport.GotoBottom()
			default:
				var cmd tea.Cmd
				a.result, cmd = a.result.Update(msg)
				return a, cmd
			}
		}

	case RunMsg:
		// Execute the tool in a goroutine and return result
		return a, runTool(msg)

	case toolResultMsg:
		a.result = NewResultModel(msg.tool, msg.output, msg.err, a.width, a.height)
		a.state = stateResult
		return a, nil
	}

	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading…"
	}

	var content string
	switch a.state {
	case stateMenu:
		content = a.menu.View()
	case stateInput:
		content = a.input.View()
	case stateResult:
		content = a.result.View()
	}

	// Wrap in the outer border
	return appBorderStyle.
		Width(a.width - 2).
		Height(a.height - 2).
		Render(content)
}

// ── async tool execution ──────────────────────────────────────────────────────

type toolResultMsg struct {
	tool   *tools.Tool
	output string
	err    string
}

func runTool(msg RunMsg) tea.Cmd {
	return func() tea.Msg {
		out, err := msg.Tool.Run(msg.Inputs)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		return toolResultMsg{tool: msg.Tool, output: out, err: errStr}
	}
}

// ── helper used by input.go View ─────────────────────────────────────────────

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth < 3 {
		return strings.Repeat(".", maxWidth)
	}
	return string(runes[:maxWidth-3]) + fmt.Sprintf("%-3s", "...")
}
