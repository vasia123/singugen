package memory

import (
	"fmt"
	"regexp"
)

var validNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidateName checks that a memory name is safe for filesystem use.
// Names must be lowercase alphanumeric with hyphens/underscores, max 64 chars.
// Do not include the .md extension.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("memory: name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("memory: name too long (%d chars, max 64)", len(name))
	}
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("memory: invalid name %q (must match [a-z0-9][a-z0-9_-]*)", name)
	}
	return nil
}
