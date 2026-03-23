package selfupdate

import "testing"

func TestValidateParentPID(t *testing.T) {
	if err := validateParentPID(0); err == nil {
		t.Error("ppid=0 should fail")
	}
	if err := validateParentPID(1); err == nil {
		t.Error("ppid=1 should fail (init)")
	}
	if err := validateParentPID(100); err != nil {
		t.Errorf("ppid=100 should pass: %v", err)
	}
}
