package extensions

import (
	"path/filepath"
	"testing"
)

func TestManager_Apply(t *testing.T) {
	dir := t.TempDir()
	mcpPath := filepath.Join(dir, "mcp.json")
	skillsDir := filepath.Join(dir, "skills")

	m := NewManager(mcpPath, skillsDir, testLogger())

	cfg := ExtensionsConfig{
		MCPServers: map[string]MCPServerConfig{
			"test-server": {Type: "stdio", Command: "echo", Args: []string{"hi"}},
		},
		Skills: []SkillConfig{
			{Name: "test-skill", Description: "Test", Content: "Do things"},
		},
	}

	if err := m.Apply(cfg); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	// MCP config written.
	if m.MCPConfigPath() != mcpPath {
		t.Errorf("MCPConfigPath = %q, want %q", m.MCPConfigPath(), mcpPath)
	}

	servers := m.ListMCPServers()
	if len(servers) != 1 {
		t.Fatalf("got %d servers, want 1", len(servers))
	}

	// Skills installed.
	skills, err := m.ListSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("got %d skills, want 1", len(skills))
	}
}

func TestManager_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "mcp.json"), filepath.Join(dir, "skills"), testLogger())

	if err := m.Apply(ExtensionsConfig{}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	if len(m.ListMCPServers()) != 0 {
		t.Error("should have no servers")
	}
}
