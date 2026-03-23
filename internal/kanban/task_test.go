package kanban

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTask_Valid(t *testing.T) {
	content := `---
title: Fix memory leak
assignee: main
priority: high
created: "2026-03-23"
due: "2026-03-25"
---

Memory leak in dreaming phase when idle timeout fires.`

	task, err := ParseTask("001-fix-bug", "in-progress", content)
	if err != nil {
		t.Fatalf("ParseTask() error: %v", err)
	}
	if task.ID != "001-fix-bug" {
		t.Errorf("ID = %q", task.ID)
	}
	if task.Title != "Fix memory leak" {
		t.Errorf("Title = %q", task.Title)
	}
	if task.Assignee != "main" {
		t.Errorf("Assignee = %q", task.Assignee)
	}
	if task.Priority != "high" {
		t.Errorf("Priority = %q", task.Priority)
	}
	if task.Status != "in-progress" {
		t.Errorf("Status = %q", task.Status)
	}
	if task.Description != "Memory leak in dreaming phase when idle timeout fires." {
		t.Errorf("Description = %q", task.Description)
	}
}

func TestParseTask_NoFrontmatter(t *testing.T) {
	_, err := ParseTask("001-test", "backlog", "just text without frontmatter")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestTask_Serialize_Roundtrip(t *testing.T) {
	original := `---
title: Test task
assignee: researcher
priority: medium
created: "2026-03-23"
due: ""
---

Some description here.`

	task, err := ParseTask("002-test", "backlog", original)
	if err != nil {
		t.Fatal(err)
	}

	serialized := task.Serialize()

	task2, err := ParseTask("002-test", "backlog", serialized)
	if err != nil {
		t.Fatalf("roundtrip parse error: %v", err)
	}

	if task.Title != task2.Title {
		t.Errorf("title: %q != %q", task.Title, task2.Title)
	}
	if task.Assignee != task2.Assignee {
		t.Errorf("assignee: %q != %q", task.Assignee, task2.Assignee)
	}
	if task.Description != task2.Description {
		t.Errorf("description: %q != %q", task.Description, task2.Description)
	}
}

func TestNextID(t *testing.T) {
	dir := t.TempDir()
	for _, col := range []string{"backlog", "in-progress"} {
		os.MkdirAll(filepath.Join(dir, col), 0755)
	}

	// Empty board — first ID is 001.
	id, err := NextID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if id != "001" {
		t.Errorf("first ID = %q, want 001", id)
	}

	// Create some files.
	os.WriteFile(filepath.Join(dir, "backlog", "003-task.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "in-progress", "007-task.md"), []byte(""), 0644)

	id, err = NextID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if id != "008" {
		t.Errorf("next ID = %q, want 008", id)
	}
}
