# ADR-0002: Claude Code Stream-JSON Protocol

## Status

Accepted

## Context

SinguGen needs to communicate with Claude Code. Two options:
1. Anthropic API directly (requires API key, costs per token)
2. Claude Code CLI binary in stream-json mode (uses user's subscription)

Users already pay for Claude subscriptions. Using the CLI binary avoids
additional API costs and leverages the full Claude Code toolset.

## Decision

Use the Claude Code binary (`claude`) with NDJSON streaming:

```
claude -p --input-format stream-json --output-format stream-json \
  --verbose --dangerously-skip-permissions
```

Communication is bidirectional NDJSON over stdin/stdout:
- Input: `{"type":"user","message":{"role":"user","content":"..."}}`
- Output: system/init, assistant (text/tool_use), result (success/error)

### Testing approach

`ProcessLauncher` interface as the mock seam. `FakeLauncher` uses
`io.Pipe` for bidirectional communication, letting tests exercise
real NDJSON serialization and session state machine code.

## Consequences

- Depends on Claude Code binary being installed in the container
- Session management via `--resume` flag with session IDs
- Process lifecycle managed by Session (watchdog, graceful shutdown)
- No API key required — uses OAuth token from subscription
- `ProcessLauncher` interface enables unit testing without real process
