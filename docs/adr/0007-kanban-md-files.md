# ADR-0007: Kanban Board with MD Files

## Status

Accepted

## Context

Need task management for user and agents. Options: GitHub Projects
(external API), SQLite (custom DB), or MD files (obsidian-like).

## Decision

MD files in directory-based columns:
```
data/kanban/{backlog,in-progress,review,done}/NNN-slug.md
```

Each file has YAML frontmatter (title, assignee, priority, dates)
and markdown body for description. Moving tasks = `os.Rename`.

### Prompt injection

Active tasks are injected into each agent's system prompt:
- Assigned tasks → go to assignee's prompt
- Unassigned tasks → go to default (main) agent
- Done tasks → excluded from all prompts
- Only title + status shown; agent reads description on demand

### Why MD over GitHub Projects

- Zero external dependencies (no auth, no network)
- Fits obsidian-like memory pattern already in use
- Human-readable, git-versionable
- GitHub Projects sync can be added later as optional layer

## Consequences

- No web UI for kanban (Telegram commands only for now)
- Tasks managed via /task, /tasks, /move, /done commands
- Agents see their tasks in system prompt automatically
- Board is shared across all agents
