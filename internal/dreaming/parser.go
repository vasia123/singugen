package dreaming

import (
	"fmt"
	"strings"

	"github.com/vasis/singugen/internal/memory"
)

const (
	markerStart = "<<<MEMORY_UPDATE>>>"
	markerEnd   = "<<<END_MEMORY_UPDATE>>>"
	markerFile  = "<<<FILE:"
	markerNone  = "<<<NO_CHANGES>>>"
)

// MemoryUpdate represents a set of changes to apply to memory.
type MemoryUpdate struct {
	Files   map[string]string // name (without .md) → content
	Changed bool
}

// ParseDreamResponse extracts memory updates from Claude's response.
func ParseDreamResponse(response string) (MemoryUpdate, error) {
	response = strings.TrimSpace(response)
	if response == "" {
		return MemoryUpdate{}, fmt.Errorf("dreaming: empty response")
	}

	if strings.Contains(response, markerNone) {
		return MemoryUpdate{Changed: false}, nil
	}

	startIdx := strings.Index(response, markerStart)
	if startIdx == -1 {
		return MemoryUpdate{}, fmt.Errorf("dreaming: missing %s marker", markerStart)
	}

	endIdx := strings.Index(response, markerEnd)
	if endIdx == -1 {
		return MemoryUpdate{}, fmt.Errorf("dreaming: missing %s marker", markerEnd)
	}

	body := response[startIdx+len(markerStart) : endIdx]
	body = strings.TrimSpace(body)

	files := make(map[string]string)
	parts := strings.Split(body, markerFile)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Part looks like: "user.md>>>\ncontent here"
		closeIdx := strings.Index(part, ">>>")
		if closeIdx == -1 {
			continue
		}

		filename := strings.TrimSpace(part[:closeIdx])
		content := strings.TrimSpace(part[closeIdx+3:])

		// Remove .md suffix if present.
		name := strings.TrimSuffix(filename, ".md")

		if err := memory.ValidateName(name); err != nil {
			return MemoryUpdate{}, fmt.Errorf("dreaming: invalid file name %q: %w", name, err)
		}

		files[name] = content
	}

	if len(files) == 0 {
		return MemoryUpdate{}, fmt.Errorf("dreaming: no files found in update")
	}

	return MemoryUpdate{Files: files, Changed: true}, nil
}
