package agent

import (
	"fmt"
	"os"
	"strings"
)

// LoadSystemPrompt reads and concatenates markdown files into a single
// system prompt string. Files are separated by double newlines.
func LoadSystemPrompt(paths ...string) (string, error) {
	if len(paths) == 0 {
		return "", nil
	}

	var parts []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("load prompt %s: %w", path, err)
		}
		parts = append(parts, strings.TrimSpace(string(data)))
	}

	return strings.Join(parts, "\n\n"), nil
}
