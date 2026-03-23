# ADR-0005: Self-Modification Pipeline

## Status

Accepted

## Context

The agent needs to modify its own code, validate changes, and restart.
Claude Code already has file editing tools — we need a safety pipeline
around it: validate → commit → signal supervisor → restart.

## Decision

### Pipeline architecture

`internal/selfupdate/` provides:
1. **CommandRunner** interface for testable external command execution
2. **Validator** — runs `go build ./cmd/agent/` + `go vet` + `go test`
3. **GitOperator** — diff, commit, push, revert via git CLI
4. **Updater** — orchestrates: check diff → protected dirs → validate → commit

### Safety boundaries

- **Protected directories**: `cmd/singugen/` and `internal/supervisor/`
  cannot be modified. Checked before validation.
- **Updater doesn't restart** — caller (Telegram `/update` command)
  decides whether to signal supervisor via SIGUSR1.
- **Circuit breaker** in supervisor prevents crash loops (existing).
- **git revert** preferred over `git reset --hard` (preserves history).

### Trigger

Explicit via Telegram `/update` command. Automatic (detect uncommitted
changes after Claude edits) deferred to future phase.

## Consequences

- Agent can self-modify within safety boundaries
- All external commands testable via CommandRunner fakes
- Supervisor remains immutable and bulletproof
- Manual rollback via `/rollback` or `git revert HEAD`
- Disabled by default (`self_update.enabled: false`)
