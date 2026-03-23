package telegram

import (
	"strings"
	"testing"
)

func TestChunkText_Empty(t *testing.T) {
	chunks := ChunkText("", 4096)
	if len(chunks) != 0 {
		t.Errorf("got %d chunks for empty string, want 0", len(chunks))
	}
}

func TestChunkText_Short(t *testing.T) {
	chunks := ChunkText("hello", 4096)
	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Errorf("got %v, want [hello]", chunks)
	}
}

func TestChunkText_ExactBoundary(t *testing.T) {
	text := strings.Repeat("a", 4096)
	chunks := ChunkText(text, 4096)
	if len(chunks) != 1 {
		t.Errorf("got %d chunks for exact boundary, want 1", len(chunks))
	}
}

func TestChunkText_SplitOnNewline(t *testing.T) {
	// 3000 chars + newline + 2000 chars = 5001, should split at newline.
	part1 := strings.Repeat("a", 3000)
	part2 := strings.Repeat("b", 2000)
	text := part1 + "\n" + part2

	chunks := ChunkText(text, 4096)
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}
	if chunks[0] != part1 {
		t.Errorf("chunk[0] length = %d, want 3000", len(chunks[0]))
	}
	if chunks[1] != part2 {
		t.Errorf("chunk[1] length = %d, want 2000", len(chunks[1]))
	}
}

func TestChunkText_HardBreak(t *testing.T) {
	// No newlines — must hard break at maxLen.
	text := strings.Repeat("x", 5000)
	chunks := ChunkText(text, 4096)
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}
	if len(chunks[0]) != 4096 {
		t.Errorf("chunk[0] length = %d, want 4096", len(chunks[0]))
	}
	if len(chunks[1]) != 904 {
		t.Errorf("chunk[1] length = %d, want 904", len(chunks[1]))
	}
}

func TestChunkText_MultiChunk(t *testing.T) {
	text := strings.Repeat("a", 10000)
	chunks := ChunkText(text, 4096)
	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3", len(chunks))
	}

	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	if total != 10000 {
		t.Errorf("total length = %d, want 10000", total)
	}
}

func TestChunkText_PreferNewlineInSecondHalf(t *testing.T) {
	// Newline at position 1000 (too early) and 3500 (good split point).
	part1 := strings.Repeat("a", 1000)
	part2 := strings.Repeat("b", 2500)
	part3 := strings.Repeat("c", 2000)
	text := part1 + "\n" + part2 + "\n" + part3

	chunks := ChunkText(text, 4096)
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}
	// Should split at the second newline (position 3501).
	if len(chunks[0]) != 3501 {
		t.Errorf("chunk[0] length = %d, want 3501 (split at second newline)", len(chunks[0]))
	}
}
