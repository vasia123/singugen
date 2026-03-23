package selfupdate

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ValidationResult holds the outcome of build/vet/test validation.
type ValidationResult struct {
	OK       bool
	Output   string
	Duration time.Duration
}

// Validator runs Go build/vet/test on the project.
type Validator struct {
	dir    string
	runner CommandRunner
}

// NewValidator creates a Validator for the given project directory.
func NewValidator(dir string, runner CommandRunner) *Validator {
	return &Validator{dir: dir, runner: runner}
}

// Validate runs go build, go vet, and go test sequentially.
// Stops on first failure.
func (v *Validator) Validate(ctx context.Context) (ValidationResult, error) {
	start := time.Now()

	steps := []struct {
		name string
		args []string
	}{
		{"build", []string{"build", "./cmd/agent/"}},
		{"vet", []string{"vet", "./..."}},
		{"test", []string{"test", "./..."}},
	}

	for _, step := range steps {
		output, err := v.runner.Run(ctx, v.dir, "go", step.args...)
		if err != nil {
			return ValidationResult{
				OK:       false,
				Output:   fmt.Sprintf("%s failed: %s\n%s", step.name, err, output),
				Duration: time.Since(start),
			}, nil
		}
	}

	return ValidationResult{
		OK:       true,
		Output:   "all checks passed",
		Duration: time.Since(start),
	}, nil
}

// CheckProtectedDirs verifies no changed file touches protected directories.
func CheckProtectedDirs(changedFiles []string, protected []string) error {
	if len(protected) == 0 {
		return nil
	}

	for _, file := range changedFiles {
		for _, dir := range protected {
			if strings.HasPrefix(file, dir+"/") || file == dir {
				return fmt.Errorf("selfupdate: protected directory modified: %s (in %s)", file, dir)
			}
		}
	}

	return nil
}
