package extensions

import (
	"path/filepath"
	"testing"
)

func TestWriteAndReadMCPConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	servers := map[string]MCPServerConfig{
		"web-search": {
			Type:    "stdio",
			Command: "node",
			Args:    []string{"/app/search.mjs"},
			Env:     map[string]string{"API_KEY": "test"},
		},
		"db": {
			Type: "sse",
			URL:  "http://localhost:3000/mcp",
		},
	}

	if err := WriteMCPConfig(path, servers); err != nil {
		t.Fatalf("WriteMCPConfig() error: %v", err)
	}

	cfg, err := ReadMCPConfig(path)
	if err != nil {
		t.Fatalf("ReadMCPConfig() error: %v", err)
	}

	if len(cfg.MCPServers) != 2 {
		t.Fatalf("got %d servers, want 2", len(cfg.MCPServers))
	}

	ws := cfg.MCPServers["web-search"]
	if ws.Command != "node" || ws.Type != "stdio" {
		t.Errorf("web-search = %+v", ws)
	}
	if ws.Env["API_KEY"] != "test" {
		t.Errorf("env = %v", ws.Env)
	}

	db := cfg.MCPServers["db"]
	if db.URL != "http://localhost:3000/mcp" {
		t.Errorf("db url = %q", db.URL)
	}
}

func TestWriteMCPConfig_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	if err := WriteMCPConfig(path, nil); err != nil {
		t.Fatalf("WriteMCPConfig() error: %v", err)
	}

	cfg, err := ReadMCPConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MCPServers) != 0 {
		t.Errorf("got %d servers, want 0", len(cfg.MCPServers))
	}
}
