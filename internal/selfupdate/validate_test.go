package selfupdate

import (
	"context"
	"fmt"
	"testing"
)

func TestValidator_AllPass(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"go build": {output: []byte(""), err: nil},
		"go vet":   {output: []byte(""), err: nil},
		"go test":  {output: []byte("ok"), err: nil},
	}}
	v := NewValidator("/project", runner)

	result, err := v.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if !result.OK {
		t.Errorf("OK = false, want true. Output: %s", result.Output)
	}
}

func TestValidator_BuildFails(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"go build": {output: []byte("compile error"), err: fmt.Errorf("exit 1")},
	}}
	v := NewValidator("/project", runner)

	result, err := v.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if result.OK {
		t.Error("OK = true, want false (build failed)")
	}
	if result.Output == "" {
		t.Error("output should contain error details")
	}
}

func TestValidator_VetFails(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"go build": {output: []byte(""), err: nil},
		"go vet":   {output: []byte("vet warning"), err: fmt.Errorf("exit 1")},
	}}
	v := NewValidator("/project", runner)

	result, _ := v.Validate(context.Background())
	if result.OK {
		t.Error("OK = true, want false (vet failed)")
	}
}

func TestValidator_TestFails(t *testing.T) {
	runner := &fakeRunner{results: map[string]runResult{
		"go build": {output: []byte(""), err: nil},
		"go vet":   {output: []byte(""), err: nil},
		"go test":  {output: []byte("FAIL"), err: fmt.Errorf("exit 1")},
	}}
	v := NewValidator("/project", runner)

	result, _ := v.Validate(context.Background())
	if result.OK {
		t.Error("OK = true, want false (test failed)")
	}
}

func TestCheckProtectedDirs(t *testing.T) {
	protected := []string{"cmd/singugen", "internal/supervisor"}

	// Safe changes.
	err := CheckProtectedDirs([]string{"internal/agent/agent.go", "cmd/agent/main.go"}, protected)
	if err != nil {
		t.Errorf("safe files should pass: %v", err)
	}

	// Violation.
	err = CheckProtectedDirs([]string{"cmd/singugen/main.go"}, protected)
	if err == nil {
		t.Error("modifying supervisor should fail")
	}

	err = CheckProtectedDirs([]string{"internal/supervisor/supervisor.go"}, protected)
	if err == nil {
		t.Error("modifying supervisor package should fail")
	}

	// Empty list = no protection.
	err = CheckProtectedDirs([]string{"cmd/singugen/main.go"}, nil)
	if err != nil {
		t.Errorf("empty protected list should allow all: %v", err)
	}
}
