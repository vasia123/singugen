---
name: project-vision
description: SinguGen core architecture decisions and vision - self-evolving AI agent wrapper around Claude Code
type: project
---

**Vision:** Self-evolving AI wrapper around Claude Code. One main agent that helps user daily, grows skills, and can spawn sub-agents as "team members."

**Key architecture decisions:**
- Claude Code binary in stream-json mode (leverages user subscription, no API costs)
- Go-first, TypeScript only if Go can't solve it
- Supervisor process manages main agent (restarts after self-modification)
- Agent can modify ALL its own code, rebuild, restart
- Obsidian-like memory: per-agent MD files, loaded into context
- "Dreaming" phase: memory reorganization, disable unused MCP/skills, archive
- MCP and skills installed by agent itself with security analysis
- Telegram Bot for chat + WebApp for rich UI (kanban, etc.)
- Kanban board: either GitHub Projects or custom WebApp implementation
- Docker deployment, ARM-compatible (Orange Pi 3 8GB)
- Agents are long-lived, communicate via channels/stream-json (reference: Praktor)
- Security-first for external skill/MCP installation

**Why:** User wants an AI companion that starts "naked" and grows capabilities organically through interaction.

**How to apply:** Every design choice should favor autonomy, self-modification capability, and minimal initial footprint. Start simple, let the agent evolve.
