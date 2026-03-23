# ADR-0001: Supervisor-Child Architecture

## Status

Accepted

## Context

SinguGen is a self-evolving AI agent that can modify its own code,
rebuild, and restart. This creates a fundamental safety problem:
if the agent can modify everything, a bad change can brick the system
with no recovery path.

## Decision

Split the system into two separate binaries:

1. **Supervisor** (`cmd/singugen/`) — immutable process manager
2. **Agent** (`cmd/agent/`) — the actual AI agent, modifiable by itself

The supervisor:
- Launches the agent as a child process via `os/exec`
- Monitors the child and restarts it on exit
- Accepts SIGUSR1 to restart the child (used after self-update)
- Implements a circuit breaker to prevent crash loops
- Sends SIGTERM (not SIGKILL) for graceful shutdown with 5s grace period

The agent:
- Can modify any code including its own `cmd/agent/main.go`
- Cannot modify `cmd/singugen/` or `internal/supervisor/`
- Signals the supervisor via SIGUSR1 after successful self-update

## Consequences

- Recovery is always possible: even if the agent bricks itself,
  the supervisor keeps running and can be used to deploy a rollback
- Circuit breaker (5 restarts in 2 minutes) prevents infinite crash loops
- Two separate build targets, both included in the Docker image
- Slightly more complex deployment (two binaries instead of one)
