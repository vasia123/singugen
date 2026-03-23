package selfupdate

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// UpdateResult holds the outcome of a self-update attempt.
type UpdateResult struct {
	Validated  bool
	Committed  bool
	CommitHash string
	Diff       string
	Output     string
}

// Updater orchestrates the self-update pipeline:
// check diff → validate protected dirs → build/test → commit → push.
type Updater struct {
	validator *Validator
	git       *GitOperator
	protected []string
	autoPush  bool
	branch    string
	logger    *slog.Logger
}

// NewUpdater creates an Updater for the given project directory.
func NewUpdater(dir string, runner CommandRunner, logger *slog.Logger) *Updater {
	return &Updater{
		validator: NewValidator(dir, runner),
		git:       NewGitOperator(dir, runner),
		logger:    logger,
	}
}

// SetProtectedDirs configures directories that cannot be modified.
func (u *Updater) SetProtectedDirs(dirs []string) {
	u.protected = dirs
}

// SetAutoPush enables automatic push after commit.
func (u *Updater) SetAutoPush(enabled bool, branch string) {
	u.autoPush = enabled
	u.branch = branch
}

// Apply runs the full self-update pipeline.
// Does NOT signal supervisor for restart — caller decides.
func (u *Updater) Apply(ctx context.Context) (UpdateResult, error) {
	// Check for changes.
	files, err := u.git.DiffFiles(ctx)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("selfupdate: diff: %w", err)
	}
	if len(files) == 0 {
		u.logger.Info("selfupdate: no changes to apply")
		return UpdateResult{}, nil
	}

	diff, _ := u.git.Diff(ctx)

	// Check protected directories.
	if err := CheckProtectedDirs(files, u.protected); err != nil {
		return UpdateResult{Diff: diff}, err
	}

	// Validate: build, vet, test.
	u.logger.Info("selfupdate: validating changes", "files", len(files))
	vr, err := u.validator.Validate(ctx)
	if err != nil {
		return UpdateResult{Diff: diff}, fmt.Errorf("selfupdate: validate: %w", err)
	}
	if !vr.OK {
		u.logger.Warn("selfupdate: validation failed", "output", vr.Output)
		return UpdateResult{Diff: diff, Validated: false, Output: vr.Output}, nil
	}

	// Commit.
	msg := fmt.Sprintf("self-update: auto-commit at %s", time.Now().Format(time.RFC3339))
	if err := u.git.Commit(ctx, msg); err != nil {
		return UpdateResult{Diff: diff, Validated: true}, fmt.Errorf("selfupdate: commit: %w", err)
	}

	hash, _ := u.git.CurrentCommit(ctx)

	u.logger.Info("selfupdate: committed", "hash", hash)

	// Push if configured.
	if u.autoPush && u.branch != "" {
		if err := u.git.Push(ctx, u.branch); err != nil {
			u.logger.Warn("selfupdate: push failed", "error", err)
		}
	}

	return UpdateResult{
		Validated:  true,
		Committed:  true,
		CommitHash: hash,
		Diff:       diff,
		Output:     vr.Output,
	}, nil
}
