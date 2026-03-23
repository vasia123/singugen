package memory

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestStore_Init_CreatesDefaults(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())

	if err := s.Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	for _, name := range defaultFiles {
		path := filepath.Join(dir, name+".md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("default file %s not created: %v", name, err)
		}
	}
}

func TestStore_Init_Idempotent(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	// Write custom content.
	s.Save("user", "custom content")

	// Init again should NOT overwrite.
	s.Init()

	entry, err := s.Load("user")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Content != "custom content" {
		t.Errorf("Init overwrote existing file: %q", entry.Content)
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	if err := s.Save("user", "Hello world"); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	entry, err := s.Load("user")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if entry.Name != "user" || entry.Content != "Hello world" {
		t.Errorf("entry = %+v, want {user, Hello world}", entry)
	}
}

func TestStore_Save_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	s.Save("custom-notes", "my notes")

	entry, err := s.Load("custom-notes")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Content != "my notes" {
		t.Errorf("content = %q, want my notes", entry.Content)
	}
}

func TestStore_LoadAll_Sorted(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	s.Save("zebra", "z content")
	s.Save("alpha", "a content")

	entries, err := s.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) < 2 {
		t.Fatalf("got %d entries, want at least 2", len(entries))
	}

	// Should be sorted alphabetically.
	for i := 1; i < len(entries); i++ {
		if entries[i].Name < entries[i-1].Name {
			t.Errorf("not sorted: %q comes after %q", entries[i].Name, entries[i-1].Name)
		}
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	s.Save("temp", "temporary")

	if err := s.Delete("temp"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := s.Load("temp")
	if err == nil {
		t.Error("Load() after Delete() should return error")
	}
}

func TestStore_Delete_NonExistent(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	err := s.Delete("nonexistent")
	if err == nil {
		t.Error("Delete nonexistent should return error")
	}
}

func TestStore_FormatForPrompt(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	s.Save("user", "I am a developer")
	s.Save("projects", "Working on SinguGen")

	prompt, err := s.FormatForPrompt()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(prompt, "## Memory: user") {
		t.Error("prompt missing user header")
	}
	if !strings.Contains(prompt, "I am a developer") {
		t.Error("prompt missing user content")
	}
	if !strings.Contains(prompt, "## Memory: projects") {
		t.Error("prompt missing projects header")
	}
}

func TestStore_FormatForPrompt_SkipsEmpty(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	// No Init — empty directory.
	os.MkdirAll(dir, 0755)

	prompt, err := s.FormatForPrompt()
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "" {
		t.Errorf("prompt should be empty for no files, got %q", prompt)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())
	s.Init()

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.Save("user", "content from goroutine")
			s.Load("user")
			s.LoadAll()
			_ = n
		}(i)
	}
	wg.Wait()
}

func TestStore_InvalidName(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, testLogger())

	if err := s.Save("../evil", "hack"); err == nil {
		t.Error("Save with path traversal should fail")
	}

	if _, err := s.Load("../evil"); err == nil {
		t.Error("Load with path traversal should fail")
	}
}
