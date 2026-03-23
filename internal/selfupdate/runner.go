package selfupdate

import (
	"context"
	"os/exec"
)

// CommandRunner abstracts external command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

// ExecCommandRunner runs commands via os/exec.
type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
