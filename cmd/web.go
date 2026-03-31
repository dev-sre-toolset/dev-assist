package cmd

import (
	"github.com/spf13/cobra"
	"github.com/dev-sre-toolset/dev-assist/internal/web"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the dev-assist web UI",
	Long: `Start an HTTP server that serves the dev-assist web interface.

Examples:
  dev-assist web                            # localhost:8080
  dev-assist web --port 9000                # localhost:9000
  dev-assist web --host 0.0.0.0 --port 80  # public on all interfaces`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		host, _ := cmd.Flags().GetString("host")
		return web.Serve(host, port)
	},
}

func init() {
	webCmd.Flags().IntP("port", "p", 8080, "port to listen on")
	webCmd.Flags().String("host", "127.0.0.1", "host to bind (use 0.0.0.0 to expose publicly)")
	rootCmd.AddCommand(webCmd)
}
