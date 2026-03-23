package selfupdate

import (
	"context"
	"fmt"
	"strings"
)

// GitOperator performs git operations on the project directory.
type GitOperator struct {
	dir    string
	runner CommandRunner
}

// NewGitOperator creates a GitOperator for the given directory.
func NewGitOperator(dir string, runner CommandRunner) *GitOperator {
	return &GitOperator{dir: dir, runner: runner}
}

// Diff returns a human-readable summary of uncommitted changes.
func (g *GitOperator) Diff(ctx context.Context) (string, error) {
	output, err := g.runner.Run(ctx, g.dir, "git", "diff", "--stat")
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// DiffFiles returns a list of changed file paths (uncommitted).
func (g *GitOperator) DiffFiles(ctx context.Context) ([]string, error) {
	output, err := g.runner.Run(ctx, g.dir, "git", "diff", "--name-only")
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only: %w", err)
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, nil
	}

	return strings.Split(text, "\n"), nil
}

// Commit stages all changes and creates a commit.
func (g *GitOperator) Commit(ctx context.Context, message string) error {
	if _, err := g.runner.Run(ctx, g.dir, "git", "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if _, err := g.runner.Run(ctx, g.dir, "git", "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// Push pushes the current branch to the remote.
func (g *GitOperator) Push(ctx context.Context, branch string) error {
	if _, err := g.runner.Run(ctx, g.dir, "git", "push", "origin", branch); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

// RevertLast creates a revert commit for HEAD.
func (g *GitOperator) RevertLast(ctx context.Context) error {
	if _, err := g.runner.Run(ctx, g.dir, "git", "revert", "HEAD", "--no-edit"); err != nil {
		return fmt.Errorf("git revert: %w", err)
	}
	return nil
}

// CurrentCommit returns the current HEAD commit hash.
func (g *GitOperator) CurrentCommit(ctx context.Context) (string, error) {
	output, err := g.runner.Run(ctx, g.dir, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
