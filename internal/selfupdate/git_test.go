package selfupdate

import (
	"context"
	"testing"
)

func TestGitOperator_Diff(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff": {output: []byte(" 2 files changed"), err: nil},
	}}
	g := NewGitOperator("/project", runner)

	diff, err := g.Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if diff != "2 files changed" {
		t.Errorf("diff = %q", diff)
	}
}

func TestGitOperator_DiffFiles(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff": {output: []byte("internal/agent/agent.go\ncmd/agent/main.go\n"), err: nil},
	}}
	g := NewGitOperator("/project", runner)

	files, err := g.DiffFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0] != "internal/agent/agent.go" {
		t.Errorf("files[0] = %q", files[0])
	}
}

func TestGitOperator_DiffFilesEmpty(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff": {output: []byte(""), err: nil},
	}}
	g := NewGitOperator("/project", runner)

	files, err := g.DiffFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files for empty diff, want 0", len(files))
	}
}

func TestGitOperator_Commit(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git add":    {output: nil, err: nil},
		"git commit": {output: nil, err: nil},
	}}
	g := NewGitOperator("/project", runner)

	err := g.Commit(context.Background(), "test commit")
	if err != nil {
		t.Fatal(err)
	}

	if !runner.Called("git add") {
		t.Error("git add not called")
	}
	if !runner.Called("git commit") {
		t.Error("git commit not called")
	}
}

func TestGitOperator_Push(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git push": {output: nil, err: nil},
	}}
	g := NewGitOperator("/project", runner)

	err := g.Push(context.Background(), "self-update")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGitOperator_RevertLast(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git revert": {output: nil, err: nil},
	}}
	g := NewGitOperator("/project", runner)

	err := g.RevertLast(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGitOperator_CurrentCommit(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git rev-parse": {output: []byte("abc123\n"), err: nil},
	}}
	g := NewGitOperator("/project", runner)

	hash, err := g.CurrentCommit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hash != "abc123" {
		t.Errorf("hash = %q, want abc123", hash)
	}
}
