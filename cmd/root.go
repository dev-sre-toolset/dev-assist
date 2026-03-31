package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/datsabk/dev-assist/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "dev-assist",
	Short: "SRE utility belt — decode, inspect, and analyse on the fly",
	Long: `dev-assist is an interactive SRE toolkit.

Run without arguments to open the interactive TUI, or pass a subcommand
for non-interactive / scriptable use.

Examples:
  dev-assist                          # open interactive TUI
  dev-assist ssl-decode --file cert.pem
  dev-assist jwt --input "eyJ..."
  dev-assist dns --host example.com --type MX`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return launchTUI()
	},
	// Don't print usage on error — we have our own error formatting.
	SilenceUsage: true,
}

// Execute is the main entry point called from main.go.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func launchTUI() error {
	// Declare the background dark so lipgloss skips its OSC 11 terminal query.
	// Without this, the query response (]11;rgb:...) leaks into the first
	// focused textarea as garbage characters.
	lipgloss.SetHasDarkBackground(true)

	p := tea.NewProgram(
		ui.NewApp(),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
