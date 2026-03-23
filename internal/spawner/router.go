package spawner

import "strings"

// ParseAgentPrefix extracts @agent_name from the beginning of a message.
// Returns the agent name (without @) and the remaining message.
// If no prefix found, returns empty name and the original message.
func ParseAgentPrefix(text string) (agentName, cleanedMsg string) {
	if !strings.HasPrefix(text, "@") || len(text) < 2 {
		return "", text
	}

	parts := strings.SplitN(text, " ", 2)
	name := strings.TrimPrefix(parts[0], "@")

	if name == "" {
		return "", text
	}

	if len(parts) > 1 {
		return name, parts[1]
	}
	return name, ""
}
