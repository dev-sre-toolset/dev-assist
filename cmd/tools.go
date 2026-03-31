package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/dev-sre-toolset/dev-assist/internal/tools"
	"github.com/dev-sre-toolset/dev-assist/internal/ui"
)

func init() {
	for _, t := range tools.Registry {
		rootCmd.AddCommand(buildCmd(t))
	}
}

// buildCmd creates a cobra.Command for a tool, deriving all flags from InputDef.
//
// Flag naming rules:
//   - Text/file inputs → single --<FlagName> flag (raw value or file path both work;
//     resolveInput handles the distinction automatically)
//   - Option toggles   → --<FlagName> with the Default pre-set
//   - Short flag       → added only when FlagShort is non-empty
//   - Fallback         → "input" for index 0, "input2/input3…" otherwise
func buildCmd(t *tools.Tool) *cobra.Command {
	values := make([]string, len(t.Inputs))

	cmd := &cobra.Command{
		Use:   useLine(t),
		Short: t.Description,
		Long:  longHelp(t),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --tui: open the interactive form for this tool
			if tui, _ := cmd.Flags().GetBool("tui"); tui {
				p := tea.NewProgram(
					ui.NewAppForTool(t),
					tea.WithAltScreen(),
					tea.WithMouseCellMotion(),
				)
				_, err := p.Run()
				return err
			}

			// Build inputs slice
			inputs := make([]string, len(t.Inputs))
			for i, def := range t.Inputs {
				v := values[i]
				if len(def.Options) > 0 && v == "" {
					v = def.Default
				}
				inputs[i] = v
			}

			// Validate required fields
			for i, def := range t.Inputs {
				if def.Required && strings.TrimSpace(inputs[i]) == "" {
					name := def.FlagName
					if name == "" {
						name = fallbackFlagName(i)
					}
					return fmt.Errorf("required flag --%s (%s) is empty", name, def.Label)
				}
			}

			out, err := t.Run(inputs)
			if err != nil {
				return fmt.Errorf("%s: %w", t.Name, err)
			}
			fmt.Print(out)
			return nil
		},
	}

	cmd.Flags().Bool("tui", false, "open interactive TUI for this tool")

	for i, def := range t.Inputs {
		name := def.FlagName
		if name == "" {
			name = fallbackFlagName(i)
		}
		short := def.FlagShort

		var usage string
		if len(def.Options) > 0 {
			usage = fmt.Sprintf("[%s]  (default: %s)", strings.Join(def.Options, "|"), def.Default)
		} else {
			usage = def.Label
			if def.AcceptsFile {
				usage += "  (raw value or file path)"
			}
		}

		dflt := def.Default // empty string for text inputs; option default for toggles
		if short != "" {
			cmd.Flags().StringVarP(&values[i], name, short, dflt, usage)
		} else {
			cmd.Flags().StringVar(&values[i], name, dflt, usage)
		}
	}

	return cmd
}

// useLine builds the Use string with all required flags shown inline.
func useLine(t *tools.Tool) string {
	var parts []string
	for _, def := range t.Inputs {
		if !def.Required {
			continue
		}
		name := def.FlagName
		if name == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("--%s <value>", name))
	}
	if len(parts) == 0 {
		return t.ID
	}
	return t.ID + "  " + strings.Join(parts, "  ")
}

// longHelp produces an informative --help body for a tool subcommand.
func longHelp(t *tools.Tool) string {
	var sb strings.Builder
	sb.WriteString(t.Description)
	sb.WriteString("\n\nCategory: ")
	sb.WriteString(t.Category)
	sb.WriteString("\n\nFlags:\n")

	for _, def := range t.Inputs {
		name := def.FlagName
		if name == "" {
			continue
		}
		short := ""
		if def.FlagShort != "" {
			short = fmt.Sprintf("-%s, ", def.FlagShort)
		}
		required := ""
		if def.Required {
			required = "  (required)"
		}
		if len(def.Options) > 0 {
			sb.WriteString(fmt.Sprintf("  %s--%s\n    %s  options: %s  default: %s\n",
				short, name, def.Label, strings.Join(def.Options, " | "), def.Default))
		} else {
			sb.WriteString(fmt.Sprintf("  %s--%s%s\n    %s\n",
				short, name, required, def.Label))
			if def.Placeholder != "" {
				sb.WriteString(fmt.Sprintf("    e.g. %s\n", def.Placeholder))
			}
		}
	}

	sb.WriteString("\nExamples:\n  dev-assist ")
	sb.WriteString(t.ID)
	sb.WriteString(" --tui\n")
	return sb.String()
}

func fallbackFlagName(i int) string {
	if i == 0 {
		return "input"
	}
	return fmt.Sprintf("input%d", i+1)
}
