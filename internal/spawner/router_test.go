package spawner

import "testing"

func TestParseAgentPrefix_WithPrefix(t *testing.T) {
	name, msg := ParseAgentPrefix("@researcher search for Go patterns")
	if name != "researcher" {
		t.Errorf("name = %q, want researcher", name)
	}
	if msg != "search for Go patterns" {
		t.Errorf("msg = %q", msg)
	}
}

func TestParseAgentPrefix_NoPrefix(t *testing.T) {
	name, msg := ParseAgentPrefix("hello world")
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
	if msg != "hello world" {
		t.Errorf("msg = %q", msg)
	}
}

func TestParseAgentPrefix_AtAlone(t *testing.T) {
	name, msg := ParseAgentPrefix("@")
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
	if msg != "@" {
		t.Errorf("msg = %q, want @", msg)
	}
}

func TestParseAgentPrefix_NameOnly(t *testing.T) {
	name, msg := ParseAgentPrefix("@coder")
	if name != "coder" {
		t.Errorf("name = %q, want coder", name)
	}
	if msg != "" {
		t.Errorf("msg = %q, want empty", msg)
	}
}

func TestParseAgentPrefix_AtInMiddle(t *testing.T) {
	name, msg := ParseAgentPrefix("hello @coder fix this")
	if name != "" {
		t.Errorf("name = %q, want empty (@ not at start)", name)
	}
	if msg != "hello @coder fix this" {
		t.Errorf("msg = %q", msg)
	}
}
