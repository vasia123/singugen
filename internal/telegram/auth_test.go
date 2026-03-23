package telegram

import "testing"

func TestIsAuthorized_Allowed(t *testing.T) {
	allow := map[int64]bool{111: true, 222: true}
	if !IsAuthorized(111, allow) {
		t.Error("user 111 should be authorized")
	}
}

func TestIsAuthorized_Denied(t *testing.T) {
	allow := map[int64]bool{111: true}
	if IsAuthorized(999, allow) {
		t.Error("user 999 should not be authorized")
	}
}

func TestIsAuthorized_EmptyAllowAll(t *testing.T) {
	if !IsAuthorized(42, nil) {
		t.Error("nil allowList should allow all")
	}
	if !IsAuthorized(42, map[int64]bool{}) {
		t.Error("empty allowList should allow all")
	}
}
