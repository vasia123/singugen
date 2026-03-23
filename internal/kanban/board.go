package kanban

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultColumns defines the standard kanban columns.
var DefaultColumns = []string{"backlog", "in-progress", "review", "done"}

// Board manages kanban tasks as MD files in directory-based columns.
type Board struct {
	dir    string
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewBoard creates a Board at the given directory.
func NewBoard(dir string, logger *slog.Logger) *Board {
	return &Board{dir: dir, logger: logger}
}

// Init creates column directories.
func (b *Board) Init() error {
	for _, col := range DefaultColumns {
		if err := os.MkdirAll(filepath.Join(b.dir, col), 0755); err != nil {
			return fmt.Errorf("kanban: create column %s: %w", col, err)
		}
	}
	return nil
}

// Add creates a new task in the backlog column.
func (b *Board) Add(title, description, assignee, priority string) (Task, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id, err := NextID(b.dir)
	if err != nil {
		return Task{}, err
	}

	slug := slugify(title)
	if slug != "" {
		id = id + "-" + slug
	}

	task := Task{
		ID:          id,
		Title:       title,
		Assignee:    assignee,
		Priority:    priority,
		Created:     time.Now().Format("2006-01-02"),
		Description: description,
		Status:      "backlog",
	}

	path := filepath.Join(b.dir, "backlog", id+".md")
	if err := os.WriteFile(path, []byte(task.Serialize()), 0644); err != nil {
		return Task{}, fmt.Errorf("kanban: write task: %w", err)
	}

	return task, nil
}

// Get finds a task by ID across all columns.
func (b *Board) Get(id string) (Task, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.findTask(id)
}

// Move transfers a task to a different column.
func (b *Board) Move(id, toColumn string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	task, err := b.findTask(id)
	if err != nil {
		return err
	}

	src := filepath.Join(b.dir, task.Status, id+".md")
	dst := filepath.Join(b.dir, toColumn, id+".md")

	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("kanban: move %s to %s: %w", id, toColumn, err)
	}

	return nil
}

// List returns tasks in a specific column.
func (b *Board) List(column string) ([]Task, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.listColumn(column)
}

// ListAll returns all tasks grouped by column.
func (b *Board) ListAll() (map[string][]Task, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[string][]Task)
	for _, col := range DefaultColumns {
		tasks, err := b.listColumn(col)
		if err != nil {
			return nil, err
		}
		result[col] = tasks
	}
	return result, nil
}

// Delete removes a task.
func (b *Board) Delete(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	task, err := b.findTask(id)
	if err != nil {
		return err
	}

	path := filepath.Join(b.dir, task.Status, id+".md")
	return os.Remove(path)
}

// FormatForAgent returns active tasks for a specific agent as prompt text.
// Assigned tasks go to their assignee. Unassigned tasks go to the default agent.
// Done tasks are excluded. Only title + status shown.
func (b *Board) FormatForAgent(agentName, defaultAgent string) (string, error) {
	all, err := b.ListAll()
	if err != nil {
		return "", err
	}

	var lines []string
	for col, tasks := range all {
		if col == "done" {
			continue
		}
		for _, task := range tasks {
			show := false
			label := ""
			if task.Assignee == agentName {
				show = true
				label = "assigned to you"
			} else if task.Assignee == "" && agentName == defaultAgent {
				show = true
				label = "unassigned"
			}
			if show {
				lines = append(lines, fmt.Sprintf("- [%s] %s [%s] (%s)", task.ID, task.Title, col, label))
			}
		}
	}

	if len(lines) == 0 {
		return "", nil
	}

	return "## Active Tasks\n" + strings.Join(lines, "\n"), nil
}

func (b *Board) findTask(id string) (Task, error) {
	for _, col := range DefaultColumns {
		path := filepath.Join(b.dir, col, id+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return ParseTask(id, col, string(data))
	}
	return Task{}, fmt.Errorf("kanban: task %s not found", id)
}

func (b *Board) listColumn(column string) ([]Task, error) {
	colDir := filepath.Join(b.dir, column)
	entries, err := os.ReadDir(colDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("kanban: list %s: %w", column, err)
	}

	var tasks []Task
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".md")
		data, err := os.ReadFile(filepath.Join(colDir, entry.Name()))
		if err != nil {
			continue
		}
		task, err := ParseTask(id, column, string(data))
		if err != nil {
			b.logger.Warn("kanban: skip malformed task", "file", entry.Name(), "error", err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func slugify(title string) string {
	title = strings.ToLower(title)
	var sb strings.Builder
	for _, r := range title {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			sb.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			sb.WriteRune('-')
		}
	}
	slug := sb.String()
	// Trim and truncate.
	slug = strings.Trim(slug, "-")
	if len(slug) > 30 {
		slug = slug[:30]
	}
	return slug
}
