package extensions

import (
	"log/slog"
)

// ExtensionsConfig holds MCP and skills configuration for an agent.
type ExtensionsConfig struct {
	MCPServers map[string]MCPServerConfig `yaml:"mcp_servers"`
	Skills     []SkillConfig              `yaml:"skills"`
}

// Manager combines MCP and skills management for an agent.
type Manager struct {
	mcpConfigPath string
	servers       map[string]MCPServerConfig
	skills        *SkillManager
	logger        *slog.Logger
}

// NewManager creates an extensions manager.
func NewManager(mcpConfigPath, skillsDir string, logger *slog.Logger) *Manager {
	return &Manager{
		mcpConfigPath: mcpConfigPath,
		servers:       make(map[string]MCPServerConfig),
		skills:        NewSkillManager(skillsDir, logger),
		logger:        logger,
	}
}

// Apply installs MCP servers and skills from config.
func (m *Manager) Apply(cfg ExtensionsConfig) error {
	// Write MCP config.
	if cfg.MCPServers != nil {
		m.servers = cfg.MCPServers
	}
	if err := WriteMCPConfig(m.mcpConfigPath, m.servers); err != nil {
		return err
	}

	// Install skills.
	for _, skill := range cfg.Skills {
		if m.skills.IsInstalled(skill.Name) {
			continue
		}
		if err := m.skills.Install(skill); err != nil {
			m.logger.Warn("failed to install skill", "name", skill.Name, "error", err)
		}
	}

	return nil
}

// MCPConfigPath returns the path to the MCP config JSON file.
func (m *Manager) MCPConfigPath() string {
	return m.mcpConfigPath
}

// ListMCPServers returns configured MCP servers.
func (m *Manager) ListMCPServers() map[string]MCPServerConfig {
	return m.servers
}

// ListSkills returns installed active skills.
func (m *Manager) ListSkills() ([]SkillConfig, error) {
	return m.skills.List()
}

// Skills returns the underlying skill manager.
func (m *Manager) Skills() *SkillManager {
	return m.skills
}
