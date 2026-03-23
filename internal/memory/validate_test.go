package memory

import (
	"strings"
	"testing"
)

func TestValidateName_Valid(t *testing.T) {
	valid := []string{"user", "projects", "skills", "my-notes", "team_log", "a1b2"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateName_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"../etc/passwd",
		"/absolute",
		"has spaces",
		"UPPER",
		".hidden",
		"user.md",
		"a/b",
		"a\\b",
		strings.Repeat("x", 65),
	}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", name)
		}
	}
}
