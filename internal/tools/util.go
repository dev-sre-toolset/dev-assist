package tools

import (
	"fmt"
	"os"
)

// readFile reads a file and returns its content as a string.
// Used by resolveInput (defined in ssl.go) and any tool that needs file I/O.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", path, err)
	}
	return string(b), nil
}
