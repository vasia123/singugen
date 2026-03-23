package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSystemPrompt(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "global.md")
	file2 := filepath.Join(dir, "agent.md")

	os.WriteFile(file1, []byte("# Global\nYou are helpful."), 0644)
	os.WriteFile(file2, []byte("# Agent\nYou are SinguGen."), 0644)

	prompt, err := LoadSystemPrompt(file1, file2)
	if err != nil {
		t.Fatalf("LoadSystemPrompt() error: %v", err)
	}

	if prompt == "" {
		t.Fatal("prompt is empty")
	}
	if !contains(prompt, "You are helpful.") {
		t.Error("prompt missing global content")
	}
	if !contains(prompt, "You are SinguGen.") {
		t.Error("prompt missing agent content")
	}
}

func TestLoadSystemPrompt_FileNotFound(t *testing.T) {
	_, err := LoadSystemPrompt("/nonexistent/file.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadSystemPrompt_Empty(t *testing.T) {
	prompt, err := LoadSystemPrompt()
	if err != nil {
		t.Fatalf("LoadSystemPrompt() error: %v", err)
	}
	if prompt != "" {
		t.Errorf("prompt = %q, want empty", prompt)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
