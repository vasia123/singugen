package selfupdate

import (
	"context"
	"fmt"
	"testing"
)

func TestUpdater_EmptyDiff(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff": {output: []byte(""), err: nil},
	}}
	u := NewUpdater("/project", runner, discardLogger())

	result, err := u.Apply(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Committed {
		t.Error("should not commit on empty diff")
	}
}

func TestUpdater_ValidationFails(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff": {output: []byte("internal/agent/agent.go\n"), err: nil},
		"go build": {output: []byte("compile error"), err: fmt.Errorf("exit 1")},
	}}
	u := NewUpdater("/project", runner, discardLogger())

	result, err := u.Apply(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Validated {
		t.Error("should not validate on build failure")
	}
	if result.Committed {
		t.Error("should not commit on validation failure")
	}
}

func TestUpdater_ProtectedDirViolation(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff": {output: []byte("cmd/singugen/main.go\n"), err: nil},
	}}
	u := NewUpdater("/project", runner, discardLogger())
	u.SetProtectedDirs([]string{"cmd/singugen", "internal/supervisor"})

	_, err := u.Apply(context.Background())
	if err == nil {
		t.Error("should fail on protected dir violation")
	}
}

func TestUpdater_FullSuccess(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff":      {output: []byte("internal/agent/agent.go\n"), err: nil},
		"go build":      {output: nil, err: nil},
		"go vet":        {output: nil, err: nil},
		"go test":       {output: nil, err: nil},
		"git add":       {output: nil, err: nil},
		"git commit":    {output: nil, err: nil},
		"git rev-parse": {output: []byte("abc123\n"), err: nil},
	}}
	u := NewUpdater("/project", runner, discardLogger())

	result, err := u.Apply(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Validated {
		t.Error("validated = false, want true")
	}
	if !result.Committed {
		t.Error("committed = false, want true")
	}
	if result.CommitHash != "abc123" {
		t.Errorf("commit hash = %q, want abc123", result.CommitHash)
	}
}

func TestUpdater_WithPush(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"git diff":      {output: []byte("file.go\n"), err: nil},
		"go build":      {output: nil, err: nil},
		"go vet":        {output: nil, err: nil},
		"go test":       {output: nil, err: nil},
		"git add":       {output: nil, err: nil},
		"git commit":    {output: nil, err: nil},
		"git rev-parse": {output: []byte("def456\n"), err: nil},
		"git push":      {output: nil, err: nil},
	}}
	u := NewUpdater("/project", runner, discardLogger())
	u.SetAutoPush(true, "self-update")

	result, err := u.Apply(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Committed {
		t.Error("should commit")
	}
	if !runner.Called("git push") {
		t.Error("git push was not called")
	}
}
