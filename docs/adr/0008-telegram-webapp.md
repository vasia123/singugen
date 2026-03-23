# ADR-0008: Telegram WebApp

## Status

Accepted

## Context

Complex interactions (kanban, memory editing, agent dashboard) need
rich UI beyond Telegram chat. Telegram Mini Apps (WebApp) provide
embedded web UI inside the Telegram app.

## Decision

### Go HTTP server with REST API

stdlib `http.ServeMux` (Go 1.22+ routing), no framework.
Endpoints expose existing Board, Memory, Pool APIs as JSON.

### Telegram initData authentication

Every WebApp session starts with initData validation:
- HMAC-SHA256 verification using bot token
- User extracted from signed payload
- AllowFrom whitelist reused from Telegram bot config
- Session token stored server-side (sync.Map)

### cloudflared tunnel for HTTPS

Quick tunnel (`cloudflared tunnel --url localhost:port`):
- Free, no account needed, works behind NAT
- URL changes on restart (agent auto-updates)
- Stable while process runs
- Tunnel is optional (config: `webapp.tunnel`)

### API-only, no frontend SPA

Frontend is future work — agent can self-build it via
self-modification (Phase 4). API is the foundation.

## Consequences

- HTTP API for kanban, memory, agents accessible from WebApp
- Auth tied to Telegram (initData), no separate user system
- cloudflared dependency optional (install separately)
- No frontend shipped — API-only MVP
- Pool.GetMemory() added for per-agent memory access
