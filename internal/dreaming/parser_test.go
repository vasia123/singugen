package dreaming

import "testing"

func TestParseDreamResponse_ValidUpdate(t *testing.T) {
	response := `Some thinking text...
<<<MEMORY_UPDATE>>>
<<<FILE:user.md>>>
I am a developer.
I like Go.
<<<FILE:projects.md>>>
Working on SinguGen.
<<<END_MEMORY_UPDATE>>>`

	update, err := ParseDreamResponse(response)
	if err != nil {
		t.Fatalf("ParseDreamResponse() error: %v", err)
	}

	if !update.Changed {
		t.Error("Changed = false, want true")
	}
	if len(update.Files) != 2 {
		t.Fatalf("got %d files, want 2", len(update.Files))
	}
	if update.Files["user"] != "I am a developer.\nI like Go." {
		t.Errorf("user content = %q", update.Files["user"])
	}
	if update.Files["projects"] != "Working on SinguGen." {
		t.Errorf("projects content = %q", update.Files["projects"])
	}
}

func TestParseDreamResponse_NoChanges(t *testing.T) {
	response := `Everything looks good.
<<<NO_CHANGES>>>`

	update, err := ParseDreamResponse(response)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if update.Changed {
		t.Error("Changed = true, want false")
	}
}

func TestParseDreamResponse_Empty(t *testing.T) {
	_, err := ParseDreamResponse("")
	if err == nil {
		t.Error("empty response should return error")
	}
}

func TestParseDreamResponse_MalformedNoEndDelimiter(t *testing.T) {
	response := `<<<MEMORY_UPDATE>>>
<<<FILE:user.md>>>
content without end`

	_, err := ParseDreamResponse(response)
	if err == nil {
		t.Error("missing END delimiter should return error")
	}
}

func TestParseDreamResponse_SingleFile(t *testing.T) {
	response := `<<<MEMORY_UPDATE>>>
<<<FILE:skills.md>>>
Go, Docker, Linux
<<<END_MEMORY_UPDATE>>>`

	update, err := ParseDreamResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(update.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(update.Files))
	}
	if update.Files["skills"] != "Go, Docker, Linux" {
		t.Errorf("skills = %q", update.Files["skills"])
	}
}

func TestParseDreamResponse_InvalidFileName(t *testing.T) {
	response := `<<<MEMORY_UPDATE>>>
<<<FILE:../evil.md>>>
hack
<<<END_MEMORY_UPDATE>>>`

	_, err := ParseDreamResponse(response)
	if err == nil {
		t.Error("path traversal in filename should return error")
	}
}
