# ADR-0004: Memory and Dreaming Design

## Status

Accepted

## Context

The agent forgets everything between sessions. Need persistent memory
(obsidian-like MD files) and a dreaming phase for memory reorganization.

Key constraint: dreaming must trigger ONLY when the agent is truly idle
(StateReady for N minutes), never during active processing.

## Decision

### Memory Store

Filesystem-based: per-agent directory with `.md` files. Default files:
user, projects, skills, journal, team. Any file name matching
`[a-z0-9][a-z0-9_-]*` is valid.

Memory is loaded into the system prompt at session start via
`FormatForPrompt()`. Not reloaded mid-session — Claude remembers
updates from conversation context.

### Idle Detection

The agent's Run loop has a `time.Timer` that:
- Starts when entering StateReady
- Stops when entering StateProcessing
- Resets after processRequest completes
- Fires only when StateReady AND queue empty

This ensures dreaming never interrupts active work.

### Dreaming Protocol

Dreamer sends a structured prompt to Claude asking to reorganize
memory files. Response uses delimiters:
```
<<<MEMORY_UPDATE>>>
<<<FILE:name.md>>>
content
<<<END_MEMORY_UPDATE>>>
```

Conservative: omitted files are NOT deleted. Only explicit updates.

### Callbacks

Agent uses `OnIdle` and `OnShutdown` callbacks — no knowledge of
dreaming/memory. Clean separation of concerns. Shutdown uses a fresh
context with timeout since the original is cancelled.

## Consequences

- Memory persists across sessions as plain MD files
- Dreaming reorganizes memory without human intervention
- Agent never dreams during processing (timer-safe design)
- No session restart needed after dreaming (conversational context)
- Shutdown dreaming ensures memory is saved before restart
