package kanban

import (
	"log/slog"
	"testing"
)

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

func TestBoard_Init(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())

	if err := b.Init(); err != nil {
		t.Fatal(err)
	}

	for _, col := range DefaultColumns {
		entries, err := b.List(col)
		if err != nil {
			t.Fatalf("List(%s) error: %v", col, err)
		}
		if len(entries) != 0 {
			t.Errorf("column %s should be empty", col)
		}
	}
}

func TestBoard_AddAndGet(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())
	b.Init()

	task, err := b.Add("Fix the bug", "Details here", "main", "high")
	if err != nil {
		t.Fatal(err)
	}

	if task.Title != "Fix the bug" {
		t.Errorf("title = %q", task.Title)
	}
	if task.Status != "backlog" {
		t.Errorf("status = %q, want backlog", task.Status)
	}
	if task.ID == "" {
		t.Error("ID is empty")
	}

	got, err := b.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Fix the bug" {
		t.Errorf("got title = %q", got.Title)
	}
}

func TestBoard_Move(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())
	b.Init()

	task, _ := b.Add("Task", "", "", "")
	if err := b.Move(task.ID, "in-progress"); err != nil {
		t.Fatal(err)
	}

	got, err := b.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "in-progress" {
		t.Errorf("status = %q, want in-progress", got.Status)
	}
}

func TestBoard_ListAll(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())
	b.Init()

	b.Add("Task 1", "", "", "")
	b.Add("Task 2", "", "", "")
	t2, _ := b.Add("Task 3", "", "", "")
	b.Move(t2.ID, "in-progress")

	all, err := b.ListAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(all["backlog"]) != 2 {
		t.Errorf("backlog has %d tasks, want 2", len(all["backlog"]))
	}
	if len(all["in-progress"]) != 1 {
		t.Errorf("in-progress has %d tasks, want 1", len(all["in-progress"]))
	}
}

func TestBoard_Delete(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())
	b.Init()

	task, _ := b.Add("To delete", "", "", "")

	if err := b.Delete(task.ID); err != nil {
		t.Fatal(err)
	}

	_, err := b.Get(task.ID)
	if err == nil {
		t.Error("Get after Delete should fail")
	}
}

func TestBoard_FormatForAgent_AssignedTasks(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())
	b.Init()

	b.Add("My task", "", "researcher", "high")
	b.Add("Other task", "", "main", "low")

	prompt, err := b.FormatForAgent("researcher", "main")
	if err != nil {
		t.Fatal(err)
	}

	if prompt == "" {
		t.Fatal("prompt is empty")
	}
	if !contains(prompt, "My task") {
		t.Error("should include assigned task")
	}
	if contains(prompt, "Other task") {
		t.Error("should NOT include task assigned to another agent")
	}
}

func TestBoard_FormatForAgent_UnassignedGoesToDefault(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())
	b.Init()

	b.Add("Unassigned task", "", "", "")

	prompt, err := b.FormatForAgent("main", "main")
	if err != nil {
		t.Fatal(err)
	}

	if !contains(prompt, "Unassigned task") {
		t.Error("default agent should see unassigned tasks")
	}

	prompt2, _ := b.FormatForAgent("researcher", "main")
	if contains(prompt2, "Unassigned task") {
		t.Error("non-default agent should NOT see unassigned tasks")
	}
}

func TestBoard_FormatForAgent_DoneExcluded(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(dir, testLogger())
	b.Init()

	task, _ := b.Add("Done task", "", "main", "")
	b.Move(task.ID, "done")

	prompt, err := b.FormatForAgent("main", "main")
	if err != nil {
		t.Fatal(err)
	}

	if contains(prompt, "Done task") {
		t.Error("done tasks should NOT appear in prompt")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
