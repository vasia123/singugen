# ADR-0003: Telegram Bot Design

## Status

Accepted

## Context

The agent needs a user-facing interface. Telegram Bot API with
long-polling is the simplest approach for MVP — no webhook setup,
no web server, works behind NAT.

## Decision

### Sender interface for testability

A `Sender` interface with 3 methods (SendMessage, EditMessage,
DeleteMessage) abstracts the Telegram API. This enables full unit
testing of the bot logic without a real Telegram connection.

Production uses `TelegoSender` wrapping the `telego` library.

### Per-request TelegramHandler

Each incoming message creates a new `Handler` implementing
`agent.MessageHandler`. This avoids shared mutable state between
concurrent requests and makes the lifecycle clear:

1. First event → send "Thinking..." status message
2. Tool use events → edit status with tool name (debounced 1/sec)
3. OnComplete → delete status, chunk and send result

### Message chunking

Telegram limits messages to 4096 characters. `ChunkText` splits
on newlines where possible, falls back to hard breaks.

## Consequences

- Bot is fully testable without Telegram API
- Per-request handlers are slightly more allocating but safer
- Status messages provide good UX during long Claude interactions
- Long-polling means no incoming webhook port needed
