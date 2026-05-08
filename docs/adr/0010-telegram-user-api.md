# ADR-0010: Telegram User API Integration

## Status

Accepted

## Context

Agent needs read access to user's full Telegram account (chats,
history, contacts) beyond what Bot API provides. Write actions
must be approved by user via inline buttons.

## Decision

### gotd/td for MTProto

Production-ready Go library for Telegram User API. Pure Go,
handles session management, authentication, reconnection.

### Coexistence with Bot API

Both APIs live side by side:
- **Bot API** (telego): commands, notifications, approval buttons
- **User API** (gotd/td): read chats, history; write with approval

### Approval flow via inline buttons

Write actions (send/delete/forward) go through ApprovalQueue:
1. Agent creates pending action with description
2. Bot sends message with [Approve ✓] [Reject ✗] inline buttons
3. User taps button → callback → execute or discard
4. Bot edits message to show result

### Authentication

One-time interactive setup: phone → code → optional 2FA.
Session persisted to `data/tg_session` (gitignored).
Subsequent runs auto-reconnect from saved session.

### Read = free, Write = approval

- Reading chats, history, contacts: no approval needed
- Sending messages, deleting, forwarding: requires user approval
- Approval via inline keyboard buttons in bot chat

## Consequences

- User must create API credentials at my.telegram.org
- One-time interactive auth required
- Session file must be protected (future: encrypt at rest)
- Rate limits stricter than Bot API
- Inline buttons available for any future approval flows
