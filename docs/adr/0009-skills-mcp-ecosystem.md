# ADR-0009: Skills & MCP Ecosystem

## Status

Accepted

## Context

Agent needs to extend capabilities with MCP servers and skills.
Claude Code supports both: `--mcp-config` for MCP servers and
`~/.claude/skills/<name>/SKILL.md` for skills.

## Decision

### MCP servers

JSON config file generated per agent, path passed via
`--mcp-config` flag. Format: `{mcpServers: {name: {type, command, args, env}}}`.

### Skills

Directory-based: each skill is a directory with SKILL.md containing
YAML frontmatter (name, description) + markdown content. Claude Code
auto-discovers them from `~/.claude/skills/`.

### Lifecycle

- Install: create directory + SKILL.md
- Disable: rename to `.disabled-<name>` (Claude won't discover)
- Enable: rename back
- Archive: move to `.archived/` directory
- Dreaming: suggest disabling unused skills

### Per-agent extensions

Each agent definition in config can have its own `extensions` block
with MCP servers and skills. Applied at spawn time.

### Security

MVP: trust by default (user explicitly installs). Future phases
will add static analysis, permission checking, and sandboxing.

## Consequences

- Agents can use MCP servers for external tools
- Skills extend Claude Code's capabilities per-agent
- Disable/archive without uninstalling (reversible)
- No marketplace integration yet (manual config)
- No security scanning (deferred)
