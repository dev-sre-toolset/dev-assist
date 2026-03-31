package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dev-sre-toolset/dev-assist/internal/tools"
)

// inputMode toggles between raw text and file-path entry.
type inputMode int

const (
	modeRaw  inputMode = iota
	modeFile           // file path; content is read at run time by resolveInput
)

// inputField holds state for one tool input slot.
type inputField struct {
	def      tools.InputDef
	mode     inputMode
	textarea textarea.Model
	textinp  textinput.Model
	optIdx   int // index into def.Options (for toggles)
}

func newInputField(def tools.InputDef, width int) inputField {
	f := inputField{def: def}

	if len(def.Options) > 0 {
		// Find default option index
		for i, o := range def.Options {
			if o == def.Default {
				f.optIdx = i
				break
			}
		}
		return f
	}

	if def.Multiline {
		ta := textarea.New()
		ta.Placeholder = def.Placeholder
		ta.SetWidth(width - 6)
		ta.SetHeight(6)
		ta.CharLimit = 0
		ta.ShowLineNumbers = false
		f.textarea = ta
	} else {
		ti := textinput.New()
		ti.Placeholder = def.Placeholder
		ti.Width = width - 6
		ti.CharLimit = 0
		f.textinp = ti
	}
	return f
}

func (f inputField) value() string {
	if len(f.def.Options) > 0 {
		if f.optIdx < len(f.def.Options) {
			return f.def.Options[f.optIdx]
		}
		return ""
	}
	if f.mode == modeFile {
		return f.textinp.Value()
	}
	if f.def.Multiline {
		return f.textarea.Value()
	}
	return f.textinp.Value()
}

// ── InputModel ────────────────────────────────────────────────────────────────

type RunMsg struct {
	Tool   *tools.Tool
	Inputs []string
}

type InputModel struct {
	tool    *tools.Tool
	fields  []inputField
	focus   int // index of focused field
	width   int
	height  int
	errMsg  string
}

func NewInputModel(t *tools.Tool, width, height int) InputModel {
	fields := make([]inputField, len(t.Inputs))
	for i, def := range t.Inputs {
		fields[i] = newInputField(def, width)
	}

	m := InputModel{
		tool:   t,
		fields: fields,
		width:  width,
		height: height,
	}
	m.focusField(0)
	return m
}

func (m *InputModel) focusField(i int) {
	// Blur all
	for j := range m.fields {
		f := &m.fields[j]
		if f.def.Multiline {
			f.textarea.Blur()
		} else {
			f.textinp.Blur()
		}
	}
	m.focus = i
	if i >= len(m.fields) {
		return
	}
	f := &m.fields[i]
	if len(f.def.Options) > 0 {
		return // option toggles don't focus
	}
	if f.mode == modeFile || !f.def.Multiline {
		f.textinp.Focus()
	} else {
		f.textarea.Focus()
	}
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Cycle focus or toggle file/raw
			next := (m.focus + 1) % len(m.fields)
			m.focusField(next)
			return m, nil

		case "shift+tab":
			prev := (m.focus - 1 + len(m.fields)) % len(m.fields)
			m.focusField(prev)
			return m, nil

		case "ctrl+f":
			// Toggle raw/file for focused field
			if m.focus < len(m.fields) {
				f := &m.fields[m.focus]
				if f.def.AcceptsFile && len(f.def.Options) == 0 {
					if f.mode == modeRaw {
						f.mode = modeFile
					} else {
						f.mode = modeRaw
					}
					// Reset inputs and re-focus
					f.textarea.Reset()
					f.textinp.Reset()
					m.focusField(m.focus)
				}
			}

		case "left", "h":
			if m.focus < len(m.fields) {
				f := &m.fields[m.focus]
				if len(f.def.Options) > 0 {
					f.optIdx = (f.optIdx - 1 + len(f.def.Options)) % len(f.def.Options)
					m.fields[m.focus] = *f
					return m, nil
				}
			}

		case "right", "l":
			if m.focus < len(m.fields) {
				f := &m.fields[m.focus]
				if len(f.def.Options) > 0 {
					f.optIdx = (f.optIdx + 1) % len(f.def.Options)
					m.fields[m.focus] = *f
					return m, nil
				}
			}

		case "ctrl+r", "ctrl+enter":
			if cmd := m.buildRunCmd(); cmd != nil {
				return m, cmd
			}
			m.errMsg = "fill all required fields (*) then press ctrl+r"
			m.focusNextEmpty()
			return m, nil

		case "enter":
			// Enter runs the tool when focused on a non-multiline field or option toggle.
			// In a multiline textarea, Enter inserts a newline as expected — use ctrl+enter there.
			if m.focus < len(m.fields) {
				f := &m.fields[m.focus]
				isMultiline := len(f.def.Options) == 0 && f.def.Multiline && f.mode == modeRaw
				if !isMultiline {
					if cmd := m.buildRunCmd(); cmd != nil {
						return m, cmd
					}
					// Required fields not yet filled — show error and move to next empty field
					m.errMsg = "fill all required fields (*) then press Enter"
					m.focusNextEmpty()
					return m, nil
				}
			}
			// Fall through: let the textarea handle Enter naturally
		}
	}

	// Forward to focused field
	if m.focus < len(m.fields) {
		f := &m.fields[m.focus]
		if len(f.def.Options) == 0 {
			if f.mode == modeFile || !f.def.Multiline {
				var cmd tea.Cmd
				f.textinp, cmd = f.textinp.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				var cmd tea.Cmd
				f.textarea, cmd = f.textarea.Update(msg)
				cmds = append(cmds, cmd)
			}
			m.fields[m.focus] = *f
		}
	}

	return m, tea.Batch(cmds...)
}

func (m InputModel) buildRunCmd() tea.Cmd {
	// Validate required fields
	for _, f := range m.fields {
		if f.def.Required && strings.TrimSpace(f.value()) == "" {
			return nil // caller should show error
		}
	}

	inputs := make([]string, len(m.fields))
	for i, f := range m.fields {
		inputs[i] = f.value()
	}

	return func() tea.Msg {
		return RunMsg{Tool: m.tool, Inputs: inputs}
	}
}

// focusNextEmpty moves focus to the first required empty field.
func (m *InputModel) focusNextEmpty() {
	for i, f := range m.fields {
		if f.def.Required && strings.TrimSpace(f.value()) == "" {
			m.focusField(i)
			return
		}
	}
}

func (m *InputModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	for i := range m.fields {
		m.fields[i].textarea.SetWidth(w - 6)
		m.fields[i].textinp.Width = w - 6
	}
}

func (m InputModel) HasRequiredValues() bool {
	for _, f := range m.fields {
		if f.def.Required && strings.TrimSpace(f.value()) == "" {
			return false
		}
	}
	return true
}

func (m InputModel) View() string {
	var sb strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	sb.WriteString("\n  " + neonBadge(" "+m.tool.Category+" ", colorPurple))
	sb.WriteString("\n  " + titleStyle.Render(m.tool.Name) + "\n")
	sb.WriteString("  " + subtitleStyle.Render(m.tool.Description) + "\n\n")

	for i, f := range m.fields {
		focused := i == m.focus

		// Label line
		label := inputLabelStyle.Render(f.def.Label)
		if f.def.Required {
			label += errorStyle.Render(" *")
		}
		if f.def.AcceptsFile && len(f.def.Options) == 0 {
			modeTag := ""
			if f.mode == modeRaw {
				modeTag = lipgloss.JoinHorizontal(lipgloss.Center,
					toggleActiveStyle.Render("raw"),
					toggleInactiveStyle.Render("file"),
				)
			} else {
				modeTag = lipgloss.JoinHorizontal(lipgloss.Center,
					toggleInactiveStyle.Render("raw"),
					toggleActiveStyle.Render("file"),
				)
			}
			label = lipgloss.JoinHorizontal(lipgloss.Center, label, "  ", modeTag)
		}
		sb.WriteString("  " + label + "\n")

		if len(f.def.Options) > 0 {
			// Option toggle row — show each option; mark selected and highlight if it's the default.
			var parts []string
			for j, opt := range f.def.Options {
				isSelected := j == f.optIdx
				isDefault := opt == f.def.Default

				label := opt
				if isDefault {
					label += " ●" // dot marks the default
				}

				if isSelected {
					parts = append(parts, toggleActiveStyle.Render(label))
				} else {
					parts = append(parts, toggleInactiveStyle.Render(label))
				}
			}
			sb.WriteString("  ")
			sb.WriteString(strings.Join(parts, "  "))
			// Contextual hint
			if focused {
				cur := f.def.Options[f.optIdx]
				hint := "  ← → to change"
				if cur == f.def.Default {
					hint += "  (using default)"
				}
				sb.WriteString(inputHintStyle.Render(hint))
			}
			sb.WriteString("\n\n")
			continue
		}

		// Text area / input
		var widget string
		var raw string
		if f.mode == modeFile || !f.def.Multiline {
			raw = f.textinp.View()
		} else {
			raw = f.textarea.View()
		}

		borderStyle := inactiveInputBorderStyle
		if focused {
			borderStyle = activeInputBorderStyle
		}
		widget = borderStyle.Width(m.width - 4).Render(raw)
		sb.WriteString(widget)
		sb.WriteString("\n\n")
	}

	if m.errMsg != "" {
		sb.WriteString("  " + errorStyle.Render("✗ "+m.errMsg) + "\n")
	}

	// Help bar
	help := strings.Join([]string{
		helpItem("tab/shift+tab", "next/prev field"),
		helpItem("enter", "run"),
		helpItem("ctrl+r", "run (always)"),
		helpItem("ctrl+f", "toggle raw/file"),
		helpItem("←→", "cycle options"),
		helpItem("esc", "back"),
	}, "  ")
	sb.WriteString("\n" + helpDescStyle.Render("  ") + help + "\n")

	return fmt.Sprintf("%s", sb.String())
}
