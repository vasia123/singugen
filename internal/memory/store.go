package memory

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var defaultFiles = []string{"user", "projects", "skills", "journal", "team"}

// Entry represents a single memory file.
type Entry struct {
	Name    string
	Content string
}

// Store manages per-agent markdown memory files on the filesystem.
type Store struct {
	dir    string
	mu     sync.RWMutex
	logger *slog.Logger
}

// New creates a memory store rooted at dir.
func New(dir string, logger *slog.Logger) *Store {
	return &Store{dir: dir, logger: logger}
}

// Init creates the memory directory and default files if they don't exist.
func (s *Store) Init() error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("memory: create dir: %w", err)
	}

	for _, name := range defaultFiles {
		path := filepath.Join(s.dir, name+".md")
		if _, err := os.Stat(path); err == nil {
			continue // already exists
		}
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			return fmt.Errorf("memory: create default %s: %w", name, err)
		}
	}

	return nil
}

// Load reads a single memory file by name.
func (s *Store) Load(name string) (Entry, error) {
	if err := ValidateName(name); err != nil {
		return Entry{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(filepath.Join(s.dir, name+".md"))
	if err != nil {
		return Entry{}, fmt.Errorf("memory: load %s: %w", name, err)
	}

	return Entry{Name: name, Content: string(data)}, nil
}

// LoadAll reads all .md files in the directory, sorted alphabetically.
func (s *Store) LoadAll() ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matches, err := filepath.Glob(filepath.Join(s.dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("memory: glob: %w", err)
	}

	sort.Strings(matches)

	var entries []Entry
	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), ".md")
		data, err := os.ReadFile(path)
		if err != nil {
			s.logger.Warn("memory: skip unreadable file", "path", path, "error", err)
			continue
		}
		entries = append(entries, Entry{Name: name, Content: string(data)})
	}

	return entries, nil
}

// Save writes content to a memory file. Creates if not exists.
func (s *Store) Save(name string, content string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, name+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("memory: save %s: %w", name, err)
	}

	return nil
}

// Delete removes a memory file.
func (s *Store) Delete(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, name+".md")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("memory: delete %s: %w", name, err)
	}

	return nil
}

// FormatForPrompt concatenates all non-empty memory entries into a
// string suitable for system prompt injection.
func (s *Store) FormatForPrompt() (string, error) {
	entries, err := s.LoadAll()
	if err != nil {
		return "", err
	}

	var parts []string
	for _, e := range entries {
		content := strings.TrimSpace(e.Content)
		if content == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("## Memory: %s\n%s", e.Name, content))
	}

	return strings.Join(parts, "\n\n"), nil
}

// Dir returns the memory directory path.
func (s *Store) Dir() string {
	return s.dir
}
