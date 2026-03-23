# ADR-0006: Multi-Agent Architecture

## Status

Accepted

## Context

The main agent needs to spawn sub-agents as "team members" with
independent sessions, memory, and specializations. Need to choose
between Docker containers (like Praktor) and goroutines.

## Decision

### Goroutines over Docker containers

All agents run as goroutines in a single process:
- Simpler deployment (single binary, supervisor unchanged)
- Lower resource overhead
- Easier debugging (shared logs)
- Trade-off: no hard isolation between agents

### Go channels for IPC

Message bus using Go channels (`internal/comms/`):
- Natural for in-process goroutines
- Type-safe, zero-copy
- Non-blocking send with buffered channels
- Upgrade path to NATS for Docker isolation later

### Agent Pool

`internal/spawner/Pool` manages agent lifecycle:
- Each agent gets own Session, Memory store, workspace
- Spawn/Stop/List at runtime via Telegram commands
- Default agent ("main") always exists, cannot be fired
- `@agent_name` prefix routing in Telegram

### Config

Agents defined in YAML (`agents.definitions`). Can also be
created at runtime via `/hire` command.

## Consequences

- Multiple agents in single process (goroutines)
- Channel-based communication, no external broker needed
- Per-agent memory isolation via directories
- Main agent is special (always exists, default route)
- `/hire`, `/fire`, `/agents` Telegram commands for management
