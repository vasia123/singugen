package kanban

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Task represents a kanban board task stored as an MD file.
type Task struct {
	ID          string `yaml:"-"`
	Title       string `yaml:"title"`
	Assignee    string `yaml:"assignee"`
	Priority    string `yaml:"priority"`
	Created     string `yaml:"created"`
	Due         string `yaml:"due"`
	Description string `yaml:"-"`
	Status      string `yaml:"-"` // derived from parent directory
}

// ParseTask parses a task from its filename, column, and file content.
func ParseTask(id, status, content string) (Task, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return Task{}, fmt.Errorf("kanban: missing frontmatter in %s", id)
	}

	var task Task
	if err := yaml.Unmarshal([]byte(parts[1]), &task); err != nil {
		return Task{}, fmt.Errorf("kanban: parse frontmatter in %s: %w", id, err)
	}

	task.ID = id
	task.Status = status
	task.Description = strings.TrimSpace(parts[2])

	return task, nil
}

// Serialize converts the task back to frontmatter + description format.
func (t Task) Serialize() string {
	header, _ := yaml.Marshal(t)
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(header)
	sb.WriteString("---\n")
	if t.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(t.Description)
	}
	return sb.String()
}

// NextID scans all columns and returns the next available ID (zero-padded 3 digits).
func NextID(boardDir string) (string, error) {
	maxID := 0

	entries, err := os.ReadDir(boardDir)
	if err != nil {
		return "001", nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		files, err := os.ReadDir(filepath.Join(boardDir, entry.Name()))
		if err != nil {
			continue
		}

		for _, f := range files {
			name := f.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}
			// Extract numeric prefix: "007-task.md" → 7
			parts := strings.SplitN(strings.TrimSuffix(name, ".md"), "-", 2)
			if len(parts) == 0 {
				continue
			}
			num, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			if num > maxID {
				maxID = num
			}
		}
	}

	return fmt.Sprintf("%03d", maxID+1), nil
}
