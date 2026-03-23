package extensions

import (
	"log/slog"
	"testing"
)

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

func TestSkillManager_InstallAndList(t *testing.T) {
	dir := t.TempDir()
	m := NewSkillManager(dir, testLogger())

	cfg := SkillConfig{
		Name:        "web-search",
		Description: "Search the web",
		Content:     "Use this skill to search...",
	}

	if err := m.Install(cfg); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	skills, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("got %d skills, want 1", len(skills))
	}
	if skills[0].Name != "web-search" {
		t.Errorf("name = %q", skills[0].Name)
	}
	if skills[0].Description != "Search the web" {
		t.Errorf("description = %q", skills[0].Description)
	}
}

func TestSkillManager_DisableAndEnable(t *testing.T) {
	dir := t.TempDir()
	m := NewSkillManager(dir, testLogger())

	m.Install(SkillConfig{Name: "test", Description: "Test", Content: "content"})

	if err := m.Disable("test"); err != nil {
		t.Fatalf("Disable() error: %v", err)
	}

	// Should not appear in active list.
	skills, _ := m.List()
	if len(skills) != 0 {
		t.Errorf("disabled skill should not appear in List()")
	}

	if err := m.Enable("test"); err != nil {
		t.Fatalf("Enable() error: %v", err)
	}

	skills, _ = m.List()
	if len(skills) != 1 {
		t.Error("re-enabled skill should appear in List()")
	}
}

func TestSkillManager_Archive(t *testing.T) {
	dir := t.TempDir()
	m := NewSkillManager(dir, testLogger())

	m.Install(SkillConfig{Name: "old", Description: "Old", Content: "..."})

	if err := m.Archive("old"); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	skills, _ := m.List()
	if len(skills) != 0 {
		t.Error("archived skill should not appear in List()")
	}

	if m.IsInstalled("old") {
		t.Error("archived skill should not be considered installed")
	}
}

func TestSkillManager_IsInstalled(t *testing.T) {
	dir := t.TempDir()
	m := NewSkillManager(dir, testLogger())

	if m.IsInstalled("nonexistent") {
		t.Error("should not be installed")
	}

	m.Install(SkillConfig{Name: "exists", Description: "X", Content: "Y"})

	if !m.IsInstalled("exists") {
		t.Error("should be installed")
	}
}
