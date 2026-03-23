package extensions

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// SkillConfig describes a Claude Code skill.
type SkillConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Content     string `yaml:"content"`
}

// SkillManager manages skills as directories with SKILL.md files.
type SkillManager struct {
	baseDir string
	logger  *slog.Logger
}

// NewSkillManager creates a SkillManager at the given directory.
func NewSkillManager(baseDir string, logger *slog.Logger) *SkillManager {
	return &SkillManager{baseDir: baseDir, logger: logger}
}

// Install creates a skill directory with SKILL.md.
func (m *SkillManager) Install(cfg SkillConfig) error {
	dir := filepath.Join(m.baseDir, cfg.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("extensions: create skill dir: %w", err)
	}

	content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", cfg.Name, cfg.Description, cfg.Content)
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("extensions: write skill: %w", err)
	}

	m.logger.Info("skill installed", "name", cfg.Name)
	return nil
}

// List returns all active (non-disabled, non-archived) skills.
func (m *SkillManager) List() ([]SkillConfig, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("extensions: list skills: %w", err)
	}

	var skills []SkillConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		skillPath := filepath.Join(m.baseDir, name, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}

		skill, err := parseSkillMD(name, string(data))
		if err != nil {
			m.logger.Warn("skip malformed skill", "name", name, "error", err)
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// Disable renames the skill directory with a .disabled prefix.
func (m *SkillManager) Disable(name string) error {
	src := filepath.Join(m.baseDir, name)
	dst := filepath.Join(m.baseDir, ".disabled-"+name)
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("extensions: disable skill %s: %w", name, err)
	}
	m.logger.Info("skill disabled", "name", name)
	return nil
}

// Enable re-enables a disabled skill.
func (m *SkillManager) Enable(name string) error {
	src := filepath.Join(m.baseDir, ".disabled-"+name)
	dst := filepath.Join(m.baseDir, name)
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("extensions: enable skill %s: %w", name, err)
	}
	m.logger.Info("skill enabled", "name", name)
	return nil
}

// Archive moves the skill to a .archived directory.
func (m *SkillManager) Archive(name string) error {
	archiveDir := filepath.Join(m.baseDir, ".archived")
	os.MkdirAll(archiveDir, 0755)

	src := filepath.Join(m.baseDir, name)
	dst := filepath.Join(archiveDir, name)
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("extensions: archive skill %s: %w", name, err)
	}
	m.logger.Info("skill archived", "name", name)
	return nil
}

// IsInstalled checks if a skill is installed and active.
func (m *SkillManager) IsInstalled(name string) bool {
	path := filepath.Join(m.baseDir, name, "SKILL.md")
	_, err := os.Stat(path)
	return err == nil
}

func parseSkillMD(name, content string) (SkillConfig, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return SkillConfig{}, fmt.Errorf("missing frontmatter")
	}

	skill := SkillConfig{Name: name}

	// Parse simple key: value from frontmatter.
	for _, line := range strings.Split(parts[1], "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, ": "); ok {
			switch k {
			case "name":
				skill.Name = v
			case "description":
				skill.Description = v
			}
		}
	}

	skill.Content = strings.TrimSpace(parts[2])
	return skill, nil
}
