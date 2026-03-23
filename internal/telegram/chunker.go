package telegram

import "strings"

// MaxMessageLen is the Telegram message character limit.
const MaxMessageLen = 4096

// ChunkText splits text into chunks no longer than maxLen bytes.
// Prefers splitting at the last newline within the chunk.
// Falls back to hard break if no newline is found.
func ChunkText(text string, maxLen int) []string {
	if len(text) == 0 {
		return nil
	}
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > maxLen {
		cutAt := maxLen

		// Look for last newline within the chunk.
		if idx := strings.LastIndex(text[:maxLen], "\n"); idx > 0 {
			cutAt = idx
		}

		chunks = append(chunks, text[:cutAt])
		text = text[cutAt:]

		// Skip the newline separator if we split on one.
		if len(text) > 0 && text[0] == '\n' {
			text = text[1:]
		}
	}

	if len(text) > 0 {
		chunks = append(chunks, text)
	}

	return chunks
}
