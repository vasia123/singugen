package extensions

import (
	"encoding/json"
	"fmt"
	"os"
)

// MCPServerConfig describes an MCP server for Claude Code.
type MCPServerConfig struct {
	Type    string            `json:"type" yaml:"type"`
	Command string            `json:"command,omitempty" yaml:"command"`
	Args    []string          `json:"args,omitempty" yaml:"args"`
	URL     string            `json:"url,omitempty" yaml:"url"`
	Env     map[string]string `json:"env,omitempty" yaml:"env"`
}

// MCPConfig is the JSON file format for --mcp-config.
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// WriteMCPConfig writes the MCP configuration to a JSON file.
func WriteMCPConfig(path string, servers map[string]MCPServerConfig) error {
	cfg := MCPConfig{MCPServers: servers}
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPServerConfig)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("extensions: marshal mcp config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("extensions: write mcp config: %w", err)
	}

	return nil
}

// ReadMCPConfig reads an MCP configuration from a JSON file.
func ReadMCPConfig(path string) (MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MCPConfig{}, fmt.Errorf("extensions: read mcp config: %w", err)
	}

	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return MCPConfig{}, fmt.Errorf("extensions: parse mcp config: %w", err)
	}

	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPServerConfig)
	}

	return cfg, nil
}
