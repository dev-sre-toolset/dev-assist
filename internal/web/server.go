package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/datsabk/dev-assist/internal/tools"
)

//go:embed static
var staticFiles embed.FS

// ansiRe strips ANSI/VT escape sequences that lipgloss injects into tool output.
//   • CSI sequences:  ESC [ ... letter   (colours, bold, reset …)
//   • OSC sequences:  ESC ] … BEL|ST    (hyperlinks, title …)
//   • ESC + anything: catches ESC \ (String Terminator) and other two-char seqs
var ansiRe = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\][^\a\x1b]*(?:\a|\x1b\\)|.)`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// ── JSON shapes ───────────────────────────────────────────────────────────────

type inputMetaJSON struct {
	Label       string   `json:"label"`
	Placeholder string   `json:"placeholder"`
	Multiline   bool     `json:"multiline"`
	Required    bool     `json:"required"`
	AcceptsFile bool     `json:"accepts_file"`
	Options     []string `json:"options"`
	Default     string   `json:"default"`
	FlagName    string   `json:"flag_name"`
}

type toolMetaJSON struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Inputs      []inputMetaJSON `json:"inputs"`
}

type runRequest struct {
	Inputs []string `json:"inputs"`
}

type runResponse struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	out := make([]toolMetaJSON, 0, len(tools.Registry))
	for _, t := range tools.Registry {
		inputs := make([]inputMetaJSON, len(t.Inputs))
		for i, def := range t.Inputs {
			opts := def.Options
			if opts == nil {
				opts = []string{}
			}
			inputs[i] = inputMetaJSON{
				Label:       def.Label,
				Placeholder: def.Placeholder,
				Multiline:   def.Multiline,
				Required:    def.Required,
				AcceptsFile: def.AcceptsFile,
				Options:     opts,
				Default:     def.Default,
				FlagName:    def.FlagName,
			}
		}
		out = append(out, toolMetaJSON{
			ID:          t.ID,
			Name:        t.Name,
			Category:    t.Category,
			Description: t.Description,
			Inputs:      inputs,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func handleRunTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tool ID from path: /api/tools/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/tools/")
	if id == "" {
		http.Error(w, "tool id required", http.StatusBadRequest)
		return
	}

	t := tools.ByID(id)
	if t == nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure inputs slice is exactly the right length.
	inputs := make([]string, len(t.Inputs))
	for i := range inputs {
		if i < len(req.Inputs) {
			inputs[i] = req.Inputs[i]
		}
	}

	output, err := t.Run(inputs)
	resp := runResponse{Output: stripANSI(output)}
	if err != nil {
		resp.Error = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

// ── Server ────────────────────────────────────────────────────────────────────

// Serve starts the HTTP server on host:port.
func Serve(host string, port int) error {
	mux := http.NewServeMux()

	// API
	mux.HandleFunc("/api/tools", handleListTools)
	mux.HandleFunc("/api/tools/", handleRunTool)

	// Static SPA — strip the embedded "static/" path prefix.
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("embed static: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	addr := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("\ndev-assist web UI\n")
	fmt.Printf("  Local:   http://127.0.0.1:%d\n", port)
	if host == "0.0.0.0" {
		fmt.Printf("  Network: http://<your-ip>:%d\n", port)
	}
	fmt.Printf("\nPress Ctrl+C to stop.\n\n")
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
